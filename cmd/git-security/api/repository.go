package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/eekwong/git-security/cmd/git-security/config"
	gh "github.com/eekwong/git-security/cmd/git-security/github"
)

type Filter struct {
	Type   string        `query:"type"`
	Field  string        `query:"field"`
	Values []interface{} `query:"values"`
	Negate bool          `query:"negate"`
}

type NameCount struct {
	Name  interface{} `bson:"_id" json:"name"`
	Count int         `bson:"count" json:"count"`
}

func (a *api) GetRepositories(c *fiber.Ctx) error {
	q := struct {
		CSV bool `query:"csv"`
	}{}
	if err := c.QueryParser(&q); err != nil {
		return err
	}

	b := struct {
		Filters []Filter `json:"filters"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	filters := bson.D{
		{
			Key:   "is_archived",
			Value: false,
		},
	}
	for _, filter := range b.Filters {
		if filter.Type == "array" {
			values := bson.A{}
			for _, v := range filter.Values {
				values = append(values, bson.M{filter.Field: v})
			}
			if filter.Negate {
				filters = append(filters, bson.E{Key: "$nor", Value: values})
			} else {
				filters = append(filters, bson.E{Key: "$or", Value: values})
			}
		} else {
			if filter.Negate {
				filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$nin": filter.Values}})
			} else {
				filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$in": filter.Values}})
			}
		}
	}
	cursor, err := a.db.Collection("repositories").Find(a.ctx, filters)
	if err != nil {
		return err
	}
	defer cursor.Close(a.ctx)

	var repos []gh.Repository
	// TODO: we can't use All if too many
	if err := cursor.All(a.ctx, &repos); err != nil {
		return err
	}

	if q.CSV {
		cursor, err := a.db.Collection("columns").Find(a.ctx, bson.D{})
		if err != nil {
			return err
		}
		var columns []config.Column
		if err := cursor.All(a.ctx, &columns); err != nil {
			return err
		}
		defer cursor.Close(a.ctx)

		records := [][]string{{
			"Repo Name",
		}}
		for _, c := range columns {
			if c.CSV {
				records[0] = append(records[0], c.Title)
			}
		}
		for _, r := range repos {
			rjson, err := json.Marshal(r)
			if err != nil {
				return err
			}
			values := []string{r.Name}
			for _, c := range columns {
				if c.CSV {
					values = append(values, gjson.GetBytes(rjson, c.Key).String())
				}
			}
			records = append(records, values)
		}
		buf := new(bytes.Buffer)
		csvWriter := csv.NewWriter(buf)
		if err := csvWriter.WriteAll(records); err != nil {
			return err
		}
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=repos.csv")
		return c.SendStream(buf)
	}
	return c.JSON(repos)
}

func (a *api) GetRepositoriesGroupBy(c *fiber.Ctx) error {
	groupBy := c.Params("groupBy")
	if groupBy == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	b := struct {
		Type    string   `json:"type"`
		Filters []Filter `json:"filters"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	filters := bson.D{
		{
			Key:   "is_archived",
			Value: false,
		},
	}
	for _, filter := range b.Filters {
		if filter.Type == "array" {
			values := bson.A{}
			for _, v := range filter.Values {
				values = append(values, bson.M{filter.Field: v})
			}
			if filter.Negate {
				filters = append(filters, bson.E{Key: "$nor", Value: values})
			} else {
				filters = append(filters, bson.E{Key: "$or", Value: values})
			}
		} else {
			if filter.Negate {
				filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$nin": filter.Values}})
			} else {
				filters = append(filters, bson.E{Key: filter.Field, Value: bson.M{"$in": filter.Values}})
			}
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

	cursor, err := a.db.Collection("repositories").Aggregate(
		a.ctx,
		mongo.Pipeline{matchStage, sortGroupByStage, groupStage, sortStage},
	)
	if err != nil {
		return err
	}

	var nameCounts []NameCount
	if err := cursor.All(a.ctx, &nameCounts); err != nil {
		return err
	}
	defer cursor.Close(a.ctx)

	// further processing to flatten the array
	if b.Type == "array" {
		flattened := make([]NameCount, 0)
		dedup := make(map[string]int)
		for _, nc := range nameCounts {
			if nc.Name != nil {
				if names, ok := nc.Name.(primitive.A); ok {
					for _, name := range names {
						name := cast.ToString(name)
						if _, ok := dedup[name]; !ok {
							dedup[name] = 0
						}
						dedup[name] += nc.Count
					}
				}
			} else {
				flattened = append(flattened, nc)
			}
		}
		for name, count := range dedup {
			flattened = append(flattened, NameCount{Name: name, Count: count})
		}
		return c.JSON(flattened)
	}

	return c.JSON(nameCounts)
}

func (a *api) AddBranchProtectionRule(c *fiber.Ctx) error {
	b := struct {
		IDs []string `json:"ids"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	cursor, err := a.db.Collection("repositories").Find(a.ctx, bson.D{
		bson.E{
			Key:   "id",
			Value: bson.M{"$in": b.IDs},
		},
	})
	if err != nil {
		return err
	}
	hasError := false
	for cursor.Next(a.ctx) {
		var repo gh.Repository
		if err := cursor.Decode(&repo); err != nil {
			return err
		}
		// check if there's already a protection branch rule
		if repo.DefaultBranchRef.BranchProtectionRule.ID == "" {
			if err := a.g.CreateBranchProtectionRule(repo.ID, repo.DefaultBranchRef.Name); err != nil {
				slog.Error(
					"error in CreateBranchProtectionRule",
					slog.String("error", err.Error()),
					slog.String("repo", repo.Name),
				)
				hasError = true
				continue
			}
			if err := a.updateRepository(&repo); err != nil {
				hasError = true
				continue
			}
		} else {
			slog.Info("ignoring CreateBranchProtectionRule due to an existing one", slog.String("repo", repo.Name))
		}
	}
	if err := cursor.Err(); err != nil {
		return err
	}
	defer cursor.Close(a.ctx)

	if hasError {
		return errors.New("encountered error in CreateBranchProtectionRule")
	}
	return c.SendStatus(200)
}

func (a *api) updateBranchProtectionRule(c *fiber.Ctx, updateField string, updateValue interface{}) error {
	b := struct {
		IDs []string `json:"ids"`
	}{}
	if err := c.BodyParser(&b); err != nil {
		return err
	}

	cursor, err := a.db.Collection("repositories").Find(a.ctx, bson.D{
		bson.E{
			Key:   "id",
			Value: bson.M{"$in": b.IDs},
		},
	})
	if err != nil {
		return err
	}
	hasError := false
	for cursor.Next(a.ctx) {
		var repo gh.Repository
		if err := cursor.Decode(&repo); err != nil {
			return err
		}
		// check if there's already a protection branch rule
		if repo.DefaultBranchRef.BranchProtectionRule.ID != "" {
			if err := a.g.UpdateBranchProtectionRule(
				repo.DefaultBranchRef.BranchProtectionRule.ID,
				updateField,
				updateValue,
			); err != nil {
				slog.Error(
					"error in CreateBranchProtectionRule",
					slog.String("error", err.Error()),
					slog.String("repo", repo.Name),
				)
				hasError = true
				continue
			}
			if err := a.updateRepository(&repo); err != nil {
				hasError = true
				continue
			}
		} else {
			slog.Info("ignoring UpdateBranchProtectionRule: not existed", slog.String("repo", repo.Name))
		}
	}
	if err := cursor.Err(); err != nil {
		return err
	}
	defer cursor.Close(a.ctx)

	if hasError {
		return errors.New("encountered error in CreateBranchProtectionRule")
	}
	return c.SendStatus(200)
}

func (a *api) RequiresPR(c *fiber.Ctx) error {
	return a.updateBranchProtectionRule(c, "RequiresApprovingReviews", true)
}

func (a *api) RequiredApprovingReviewCount(c *fiber.Ctx) error {
	return a.updateBranchProtectionRule(c, "RequiredApprovingReviewCount", 2)
}

func (a *api) DismissesStaleReviews(c *fiber.Ctx) error {
	return a.updateBranchProtectionRule(c, "DismissesStaleReviews", true)
}

func (a *api) RequiresConversationResolution(c *fiber.Ctx) error {
	return a.updateBranchProtectionRule(c, "RequiresConversationResolution", true)
}

func (a *api) AllowsForcePushes(c *fiber.Ctx) error {
	return a.updateBranchProtectionRule(c, "AllowsForcePushes", false)
}

func (a *api) AllowsDeletions(c *fiber.Ctx) error {
	return a.updateBranchProtectionRule(c, "AllowsDeletions", false)
}

func (a *api) updateRepository(repo *gh.Repository) error {

	updatedRepo, err := a.g.GetRepo(repo.Owner.Login, repo.Name)
	if err != nil {
		slog.Error(
			"error in GetRepo",
			slog.String("error", err.Error()),
			slog.String("repo", repo.Name),
		)
		return err
	}

	updatedRepo.FetchedAt = time.Now()
	filter := bson.D{{Key: "id", Value: repo.ID}}
	update := bson.D{{Key: "$set", Value: updatedRepo}}
	if _, err := a.db.Collection("repositories").
		UpdateOne(a.ctx, filter, update); err != nil {
		slog.Error(
			"error in updating the database",
			slog.String("error", err.Error()),
			slog.String("repo", repo.Name),
		)
		return err
	}
	a.broadcastMessage(*updatedRepo)
	return nil
}
