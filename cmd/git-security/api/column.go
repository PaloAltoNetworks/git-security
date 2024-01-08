package api

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/xissy/lexorank"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/eekwong/git-security/cmd/git-security/config"
)

func (a *api) GetColumns(c *fiber.Ctx) error {
	cursor, err := a.db.Collection("columns").Find(
		a.ctx,
		bson.D{},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}}),
	)
	if err != nil {
		return err
	}
	defer cursor.Close(a.ctx)
	var columns []config.Column
	if err := cursor.All(a.ctx, &columns); err != nil {
		return err
	}
	return c.JSON(columns)
}

func (a *api) CreateColumn(c *fiber.Ctx) error {
	var col config.Column
	if err := a.db.Collection("columns").FindOne(
		a.ctx,
		bson.D{},
		options.FindOne().SetSort(bson.D{{Key: "order", Value: 1}}),
	).Decode(&col); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}
	r, _ := lexorank.Rank("", col.Order)
	if _, err := a.db.Collection("columns").InsertOne(
		a.ctx,
		config.Column{
			Type:  "string",
			Width: 100,
			Order: r,
		},
	); err != nil {
		slog.Error("error in inserting a column", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}

func (a *api) ChangeColumnsOrder(c *fiber.Ctx) error {
	b := struct {
		ID   string `json:"id"`
		Prev string `json:"prev"`
		Next string `json:"next"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}
	r, _ := lexorank.Rank(b.Prev, b.Next)
	id, err := primitive.ObjectIDFromHex(b.ID)
	if err != nil {
		return err
	}
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.M{"order": r}}}
	if _, err := a.db.Collection("columns").UpdateOne(a.ctx, filter, update); err != nil {
		slog.Error("error in updating the column order", slog.String("newR", r))
		return err
	}
	return c.SendStatus(200)
}

func (a *api) UpdateColumn(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}
	var column config.Column
	if err := c.BodyParser(&column); err != nil {
		return err
	}
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: column}}
	if _, err := a.db.Collection("columns").UpdateOne(a.ctx, filter, update); err != nil {
		slog.Error("error in updating the column", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}

func (a *api) DeleteColumn(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}
	filter := bson.D{{Key: "_id", Value: id}}
	if _, err := a.db.Collection("columns").DeleteOne(a.ctx, filter); err != nil {
		slog.Error("error in deleting the column", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}
