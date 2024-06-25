package api

import (
	"log/slog"
	"time"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UsernameDuration struct {
	Username interface{} `bson:"_id" json:"username"`
	Duration int         `bson:"duration" json:"duration"`
}

func (a *api) GetUserView(c *fiber.Ctx) error {
	username, err := a.getUsernameFromSession(c)
	if err != nil {
		return err
	}

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
	username, err := a.getUsernameFromSession(c)
	if err != nil {
		return err
	}

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

func (a *api) GetLoggeds(c *fiber.Ctx) error {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	filters := bson.D{
		{Key: "start", Value: bson.M{"$gte": thirtyDaysAgo}},
	}
	matchStage := bson.D{{Key: "$match", Value: filters}}
	sortGroupByStage := bson.D{{Key: "$sort", Value: bson.D{{Key: "username", Value: 1}}}}
	groupStage := bson.D{
		{
			Key: "$group",
			Value: bson.D{
				{Key: "_id", Value: "$username"},
				{Key: "duration", Value: bson.D{{Key: "$sum", Value: "$duration"}}},
			},
		},
	}
	cursor, err := a.db.Collection("logged").Aggregate(
		a.ctx,
		mongo.Pipeline{matchStage, sortGroupByStage, groupStage},
	)
	if err != nil {
		return err
	}

	durations := []UsernameDuration{}
	if err := cursor.All(a.ctx, &durations); err != nil {
		return err
	}
	defer cursor.Close(a.ctx)

	return c.JSON(durations)
}
