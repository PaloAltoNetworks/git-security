package api

import (
	"log/slog"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (a *api) GetUserView(c *fiber.Ctx) error {
	sess, err := a.store.Get(c)
	if err != nil {
		return err
	}
	username := cast.ToString(sess.Get("username"))

	var uv config.UserView
	if err := a.db.Collection("userviews").FindOne(
		a.ctx,
		bson.D{{Key: "username", Value: username}},
	).Decode(&uv); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		} else {
			// can't find the userview
			// get all columns
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

			// create UV
			uv.Username = username
			uv.Filters = make([]config.UserViewFilter, 0)
			uv.Columns = make([]primitive.ObjectID, 0)
			for _, c := range columns {
				if c.Filter {
					uv.Filters = append(uv.Filters, config.UserViewFilter{
						ID:             c.ID,
						FilterExpanded: c.FilterExpanded,
					})
				}

				if c.Show {
					uv.Columns = append(uv.Columns, c.ID)
				}
			}

			// save it
			filter := bson.D{{Key: "username", Value: username}}
			update := bson.D{{Key: "$set", Value: uv}}
			if _, err := a.db.Collection("userviews").UpdateOne(a.ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
				slog.Error("error in updating the user view", slog.String("error", err.Error()))
				return err
			}
		}
	}
	return c.JSON(uv)
}

func (a *api) UpdateUserView(c *fiber.Ctx) error {
	sess, err := a.store.Get(c)
	if err != nil {
		return err
	}
	username := cast.ToString(sess.Get("username"))

	var uv config.UserView
	if err := c.BodyParser(&uv); err != nil {
		return err
	}
	uv.Username = username

	filter := bson.D{{Key: "username", Value: username}}
	update := bson.D{{Key: "$set", Value: uv}}
	if _, err := a.db.Collection("userviews").UpdateOne(a.ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
		slog.Error("error in updating the user view", slog.String("error", err.Error()))
		return err
	}
	return c.SendStatus(200)
}
