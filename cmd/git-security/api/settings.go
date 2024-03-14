package api

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
)

func (a *api) GetGlobalSettings(c *fiber.Ctx) error {
	gs := config.GlobalSettings{
		ScoreColors:  make([]config.ScoreColor, 0),
		ScoreWeights: make([]config.ScoreWeight, 0),
	}
	if err := a.db.Collection("globalSettings").FindOne(
		a.ctx,
		bson.D{},
	).Decode(&gs); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}
	return c.JSON(gs)
}

func (a *api) UpdateGlobalSettings(c *fiber.Ctx) error {
	var gs config.GlobalSettings
	if err := c.BodyParser(&gs); err != nil {
		return err
	}
	filter := bson.D{}
	update := bson.D{{Key: "$set", Value: gs}}
	if _, err := a.db.Collection("globalSettings").UpdateOne(a.ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
		slog.Error("error in updating the global settings", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}
