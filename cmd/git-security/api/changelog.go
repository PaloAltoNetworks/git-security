package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (a *api) GetChangelog(c *fiber.Ctx) error {
	q := struct {
		CSV bool `query:"csv"`
	}{}
	if err := c.QueryParser(&q); err != nil {
		return err
	}

	b := struct {
		Filters   []Filter `json:"filters"`
		StartDate int64    `json:"start_date"`
		EndDate   int64    `json:"end_date"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	start := time.Now().AddDate(0, 0, -30)
	if b.StartDate > 0 {
		start = time.Unix(b.StartDate, 0)
	}
	end := time.Now()
	if b.EndDate > 0 {
		end = time.Unix(b.EndDate, 0)
	}

	filters := bson.D{bson.E{Key: "created_at", Value: bson.M{"$gte": start, "$lte": end}}}
	for _, filter := range b.Filters {
		if filter.Negate {
			filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$nin": filter.Values}})
		} else {
			filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$in": filter.Values}})
		}
	}
	changelog, err := a.dbw.ReadChangelog(filters)
	if err != nil {
		return err
	}

	if q.CSV {
		records := [][]string{{
			"Repo Name", "Organization",
			"Repo Owner", "Repo Owner Contact",
			"Field", "From", "To", "Created At",
		}}
		for _, c := range changelog {
			records = append(records, []string{
				c.Name, c.Owner.Login,
				c.RepoOwner, c.RepoOwnerContact,
				c.Field, c.From, c.To, c.CreatedAt.String(),
			})
		}
		buf := new(bytes.Buffer)
		csvWriter := csv.NewWriter(buf)
		if err := csvWriter.WriteAll(records); err != nil {
			return err
		}
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=repos_changelog.csv")
		return c.SendStream(buf)
	}

	return c.JSON(changelog)
}

func (a *api) GetChangelogGroupBy(c *fiber.Ctx) error {
	groupBy := c.Params("groupBy")
	if groupBy == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	b := struct {
		Filters   []Filter `json:"filters"`
		StartDate int64    `json:"start_date"`
		EndDate   int64    `json:"end_date"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	start := time.Now().AddDate(0, 0, -30)
	if b.StartDate > 0 {
		start = time.Unix(b.StartDate, 0)
	}
	end := time.Now()
	if b.EndDate > 0 {
		end = time.Unix(b.EndDate, 0)
	}

	filters := bson.D{bson.E{Key: "created_at", Value: bson.M{"$gte": start, "$lte": end}}}
	for _, filter := range b.Filters {
		if filter.Negate {
			filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$nin": filter.Values}})
		} else {
			filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$in": filter.Values}})
		}
	}

	matchStage := bson.D{{Key: "$match", Value: filters}}
	sortGroupByStage := bson.D{{Key: "$sort", Value: bson.D{{Key: groupBy, Value: 1}}}}
	groupStage := bson.D{
		{
			Key: "$group",
			Value: bson.D{
				{Key: "_id", Value: fmt.Sprintf("$%s", groupBy)},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			},
		},
	}
	sortStage := bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}}

	cursor, err := a.db.Collection("changelog").Aggregate(
		a.ctx,
		mongo.Pipeline{matchStage, sortGroupByStage, groupStage, sortStage},
	)
	if err != nil {
		return err
	}

	nameCounts := []NameCount{}
	if err := cursor.All(a.ctx, &nameCounts); err != nil {
		return err
	}
	defer cursor.Close(a.ctx)

	return c.JSON(nameCounts)
}
