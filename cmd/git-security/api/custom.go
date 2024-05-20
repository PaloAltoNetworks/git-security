package api

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/security"
)

func (a *api) GetCustoms(c *fiber.Ctx) error {
	cursor, err := a.db.Collection("customs").Find(a.ctx, bson.D{})
	if err != nil {
		return err
	}
	defer cursor.Close(a.ctx)
	customs := []config.Custom{}
	if err := cursor.All(a.ctx, &customs); err != nil {
		return err
	}

	for idx := range customs {
		for idy := range customs[idx].Envs {
			customs[idx].Envs[idy].Value, err = security.Decrypt(customs[idx].Envs[idy].Value, a.key)
			if err != nil {
				return err
			}
		}
	}

	slices.Reverse(customs)
	return c.JSON(customs)
}

func (a *api) CreateCustom(c *fiber.Ctx) error {
	if _, err := a.db.Collection("customs").InsertOne(
		a.ctx,
		config.Custom{
			ValueType:    "string",
			DefaultValue: "",
			ErrorValue:   "",
		},
	); err != nil {
		slog.Error("error in inserting a custom", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}

func (a *api) UpdateCustom(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}

	var old config.Custom
	if err := a.db.Collection("customs").FindOne(
		a.ctx,
		bson.D{{Key: "_id", Value: id}},
	).Decode(&old); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}
	oldField := old.Field

	var custom config.Custom
	if err := c.BodyParser(&custom); err != nil {
		return err
	}

	// encrypt
	for idx := range custom.Envs {
		custom.Envs[idx].Value, err = security.Encrypt(custom.Envs[idx].Value, a.key)
		if err != nil {
			return err
		}
	}

	// create new data for default
	if oldField != custom.Field {
		// update all the null customs first
		if _, err := a.db.Collection("repositories").UpdateMany(
			a.ctx,
			bson.D{
				{Key: "customs", Value: nil},
			},
			bson.D{
				{
					Key:   "$set",
					Value: bson.D{{Key: "customs", Value: make(map[string]interface{})}},
				},
			},
		); err != nil {
			slog.Error(
				"error in adding empty customs for repos",
				slog.String("error", err.Error()),
			)
			return err
		}

		if _, err := a.db.Collection("repositories").UpdateMany(
			a.ctx,
			bson.D{},
			bson.D{
				{
					Key:   "$set",
					Value: bson.D{{Key: fmt.Sprintf("customs.%s", custom.Field), Value: custom.DefaultValue}},
				},
			},
		); err != nil {
			slog.Error(
				"error in adding new field for repos",
				slog.String("error", err.Error()),
				slog.String("field", custom.Field),
			)
			return err
		}
	}

	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: custom}}
	if _, err := a.db.Collection("customs").UpdateOne(a.ctx, filter, update); err != nil {
		slog.Error("error in updating the custom", slog.String("error", err.Error()))
		return err
	}

	if oldField != custom.Field && oldField != "" {
		// delete the old data
		if _, err := a.db.Collection("repositories").UpdateMany(
			a.ctx,
			bson.D{},
			bson.D{{Key: "$unset", Value: bson.D{{Key: fmt.Sprintf("customs.%s", oldField), Value: ""}}}},
		); err != nil {
			slog.Error(
				"error in removing field from repos",
				slog.String("error", err.Error()),
				slog.String("field", oldField),
			)
			return err
		}
	}

	return c.SendStatus(200)
}

func (a *api) DeleteCustom(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}

	var old config.Custom
	if err := a.db.Collection("customs").FindOne(
		a.ctx,
		bson.D{{Key: "_id", Value: id}},
	).Decode(&old); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}

	filter := bson.D{{Key: "_id", Value: id}}
	if _, err := a.db.Collection("customs").DeleteOne(a.ctx, filter); err != nil {
		slog.Error("error in deleting the custom", slog.String("error", err.Error()))
		return err
	}

	// delete the old data
	if old.Field != "" {
		if _, err := a.db.Collection("repositories").UpdateMany(
			a.ctx,
			bson.D{},
			bson.D{{Key: "$unset", Value: bson.D{{Key: fmt.Sprintf("customs.%s", old.Field), Value: ""}}}},
		); err != nil {
			slog.Error(
				"error in removing field from customs",
				slog.String("error", err.Error()),
				slog.String("field", old.Field),
			)
			return err
		}
	}

	return c.SendStatus(200)
}
