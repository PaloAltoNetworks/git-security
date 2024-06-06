package api

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
)

func (a *api) GetOwners(c *fiber.Ctx) error {
	cursor, err := a.db.Collection("owners").Find(
		a.ctx,
		bson.D{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return err
	}
	defer cursor.Close(a.ctx)
	owners := []config.Owner{}
	if err := cursor.All(a.ctx, &owners); err != nil {
		return err
	}
	return c.JSON(owners)
}

func (a *api) CreateOwner(c *fiber.Ctx) error {
	if _, err := a.db.Collection("owners").InsertOne(
		a.ctx,
		config.Owner{
			Name: strconv.FormatInt(time.Now().UnixMilli(), 10),
		},
	); err != nil {
		slog.Error("error in inserting an owner", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}

func (a *api) UpdateOwner(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}
	var owner config.Owner
	if err := c.BodyParser(&owner); err != nil {
		return err
	}
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: owner}}
	if _, err := a.db.Collection("owners").UpdateOne(a.ctx, filter, update); err != nil {
		slog.Error("error in updating the owner", slog.String("error", err.Error()))
		return err
	}

	// update all the repos with the corresponding owner
	filter = bson.D{{Key: "repo_owner_id", Value: id}}
	update = bson.D{{Key: "$set", Value: bson.D{
		{Key: "repo_owner", Value: owner.Name},
		{Key: "repo_owner_contact", Value: owner.Contact},
	}}}

	repos, err := a.dbw.UpdateRepositories(filter, update)
	if err != nil {
		return err
	}
	for _, r := range repos {
		a.broadcastMessage(*r)
	}

	return c.SendStatus(200)
}

func (a *api) DeleteOwner(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}
	filter := bson.D{{Key: "_id", Value: id}}
	if _, err := a.db.Collection("owners").DeleteOne(a.ctx, filter); err != nil {
		slog.Error("error in deleting the owner", slog.String("error", err.Error()))
		return err
	}

	// unset all the repos with the deleted owner
	filter = bson.D{{Key: "repo_owner_id", Value: id}}
	update := bson.D{{Key: "$unset", Value: bson.D{
		{Key: "repo_owner_id", Value: ""},
		{Key: "repo_owner", Value: ""},
		{Key: "repo_owner_contact", Value: ""},
	}}}

	repos, err := a.dbw.UpdateRepositories(filter, update)
	if err != nil {
		return err
	}
	for _, r := range repos {
		a.broadcastMessage(*r)
	}

	return c.SendStatus(200)
}
