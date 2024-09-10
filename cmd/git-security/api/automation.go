package api

import (
	"log/slog"
	"slices"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/security"
)

func (a *api) GetAutomations(c *fiber.Ctx) error {
	cursor, err := a.db.Collection("automations").Find(a.ctx, bson.D{})
	if err != nil {
		return err
	}
	defer cursor.Close(a.ctx)
	automations := []config.Automation{}
	if err := cursor.All(a.ctx, &automations); err != nil {
		return err
	}

	for idx := range automations {
		for idy := range automations[idx].Envs {
			automations[idx].Envs[idy].Value, err = security.Decrypt(automations[idx].Envs[idy].Value, a.key)
			if err != nil {
				return err
			}
		}
	}

	slices.Reverse(automations)
	return c.JSON(automations)
}

func (a *api) CreateAutomation(c *fiber.Ctx) error {
	if _, err := a.db.Collection("automations").InsertOne(
		a.ctx,
		config.Automation{},
	); err != nil {
		slog.Error("error in inserting a automation", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}

func (a *api) UpdateAutomation(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}

	var automation config.Automation
	if err := c.BodyParser(&automation); err != nil {
		return err
	}

	// encrypt
	for idx := range automation.Envs {
		automation.Envs[idx].Value, err = security.Encrypt(automation.Envs[idx].Value, a.key)
		if err != nil {
			return err
		}
	}

	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$set", Value: automation}}
	if _, err := a.db.Collection("automations").UpdateOne(a.ctx, filter, update); err != nil {
		slog.Error("error in updating the automation", slog.String("error", err.Error()))
		return err
	}

	return c.SendStatus(200)
}

func (a *api) DeleteAutomation(c *fiber.Ctx) error {
	_id := c.Params("id")
	if _id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return err
	}

	filter := bson.D{{Key: "_id", Value: id}}
	if _, err := a.db.Collection("automations").DeleteOne(a.ctx, filter); err != nil {
		slog.Error("error in deleting the automation", slog.String("error", err.Error()))
		return err
	}

	return c.SendStatus(200)
}
