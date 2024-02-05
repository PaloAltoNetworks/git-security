package api

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"go.mongodb.org/mongo-driver/mongo"

	gh "github.com/eekwong/git-security/cmd/git-security/github"
)

type api struct {
	ctx context.Context
	db  *mongo.Database
	g   gh.GitHub
	key []byte
}

func NewFiberApp(
	ctx context.Context,
	db *mongo.Database,
	g gh.GitHub,
	key []byte,
	adminUsername string,
	adminPassword string,
) *fiber.App {
	app := fiber.New()
	app.Use(compress.New())

	a := api{
		ctx: ctx,
		db:  db,
		g:   g,
		key: key,
	}

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	app.Get("/logout", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusUnauthorized)
	})
	app.Use(basicauth.New(basicauth.Config{
		Users: map[string]string{
			adminUsername: adminPassword,
		},
	}))

	apiRoute := app.Group("/api")
	v1 := apiRoute.Group("/v1")
	v1.Delete("/column/:id", a.DeleteColumn)
	v1.Delete("/custom/:id", a.DeleteCustom)
	v1.Get("/columns", a.GetColumns)
	v1.Get("/customs", a.GetCustoms)
	v1.Post("/columns", a.CreateColumn)
	v1.Post("/customs", a.CreateCustom)
	v1.Post("/columns/order", a.ChangeColumnsOrder)
	v1.Post("/repos", a.GetRepositories)
	v1.Post("/repos/:groupBy", a.GetRepositoriesGroupBy)
	v1.Post("/repos/action/add-branch-protection-rule", a.AddBranchProtectionRule)
	v1.Post("/repos/action/requires-pr", a.RequiresPR)
	v1.Post("/repos/action/required-approving-review-count", a.RequiredApprovingReviewCount)
	v1.Post("/repos/action/dismisses-stale-reviews", a.DismissesStaleReviews)
	v1.Post("/repos/action/requires-conversation-resolution", a.RequiresConversationResolution)
	v1.Post("/repos/action/allows-force-pushes", a.AllowsForcePushes)
	v1.Post("/repos/action/allows-deletions", a.AllowsDeletions)
	v1.Put("/column/:id", a.UpdateColumn)
	v1.Put("/custom/:id", a.UpdateCustom)

	currentDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	filesDir := filepath.Join(currentDir, "ui")
	app.Static("/", filesDir)

	return app
}
