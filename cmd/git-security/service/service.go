package service

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/FerretDB/FerretDB/ferretdb"
	flag "github.com/eekwong/go-common-flags"
	"github.com/eekwong/go-interruptible-service"
	"github.com/xissy/lexorank"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/eekwong/git-security/cmd/git-security/api"
	"github.com/eekwong/git-security/cmd/git-security/config"
	gh "github.com/eekwong/git-security/cmd/git-security/github"
)

type Opts struct {
	GitHub        *flag.GitHubOpts
	Http          *flag.HttpOpts
	Https         *flag.HttpsOpts
	Postgres      *flag.PostgresOpts
	Mongo         *flag.MongoOpts
	Key           string
	CACert        string
	DB            string
	AdminUsername string
	AdminPassword string
}

type GitSecurityApp struct {
	interruptible.Service
	ctx  context.Context
	opts *Opts
	db   *mongo.Database
	g    gh.GitHub
	key  []byte
}

func New(opts *Opts) *GitSecurityApp {
	return &GitSecurityApp{
		opts: opts,
	}
}

func (app *GitSecurityApp) Run() (interruptible.Stop, error) {
	if app.opts.DB != "sqlite" && app.opts.DB != "pg" && app.opts.DB != "mongo" {
		return nil, fmt.Errorf("error in the db argument: %s", app.opts.DB)
	}
	slog.Info("starting git-security", slog.String("db", app.opts.DB))

	ctx, cancel := context.WithCancel(context.Background())
	app.ctx = ctx

	var wg sync.WaitGroup

	uri := app.opts.Mongo.GetURI()

	if app.opts.DB != "mongo" {
		os.Mkdir(app.opts.DB, os.ModePerm)
		f, err := ferretdb.New(&ferretdb.Config{
			Listener: ferretdb.ListenerConfig{
				TCP: "127.0.0.1:27017",
			},
			Handler:   app.opts.DB,
			SQLiteURL: "file:sqlite/",
			PostgreSQLURL: fmt.Sprintf(
				"postgres://%s:%s@%s:%d/%s",
				app.opts.Postgres.PostgresUsername,
				app.opts.Postgres.PostgresPassword,
				app.opts.Postgres.PostgresHost,
				app.opts.Postgres.PostgresPort,
				app.opts.Postgres.PostgresDBName,
			),
		})
		if err != nil {
			cancel()
			return nil, err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := f.Run(ctx); err != nil {
				slog.Error("error in running FerretDB", slog.String("error", err.Error()))
			}
		}()

		uri = f.MongoDBURI()
	}

	m, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		cancel()
		return nil, err
	}

	app.db = m.Database("public")

	// create indices
	if err := app.createIndices(ctx); err != nil {
		cancel()
		return nil, err
	}

	// create default columns
	app.createDefaultColumns(ctx)

	// setup github clients
	app.g, err = gh.New(ctx, app.opts.GitHub.Host, app.opts.GitHub.PAT, app.opts.CACert)
	if err != nil {
		cancel()
		return nil, err
	}

	app.key, err = base64.StdEncoding.DecodeString(app.opts.Key)
	if err != nil {
		cancel()
		return nil, err
	}

	// web server
	fiberApp := api.NewFiberApp(ctx, app.db, app.g, app.key, app.opts.AdminUsername, app.opts.AdminPassword)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.opts.Https.IsEnabled() {
			// Create tls certificate
			cer, err := tls.LoadX509KeyPair(
				app.opts.Https.HttpsSSLCertLocation,
				app.opts.Https.HttpsSSLKeyLocation,
			)
			if err != nil {
				slog.Error("error in tls.LoadX509KeyPair", slog.String("error", err.Error()))
			}

			config := &tls.Config{Certificates: []tls.Certificate{cer}}

			// Create custom listener
			ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", app.opts.Https.HttpsPort), config)
			if err != nil {
				slog.Error("error in tls.Listen", slog.String("error", err.Error()))
			}

			// Start server with https/ssl enabled
			if err := fiberApp.Listener(ln); err != nil {
				slog.Error("error in fiberApp.Listen", slog.String("error", err.Error()))
			}
		} else {
			if err := fiberApp.Listen(fmt.Sprintf(":%d", app.opts.Http.HttpPort)); err != nil {
				slog.Error("fiberApp.listen", slog.String("error", err.Error()))
			}
		}
	}()

	// fetch github repos
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := app.fetch(); err != nil {
			slog.Error("error in app.fetch()", slog.String("error", err.Error()))
		}

	loop:
		for {
			select {
			case <-app.ctx.Done():
				break loop
			case <-time.After(5 * time.Minute):
				if err := app.fetch(); err != nil {
					slog.Error("error in app.fetch()", slog.String("error", err.Error()))
				}
			}
		}
	}()

	// run custom logics
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := app.runCustom(); err != nil {
			slog.Error("error in app.runCustom()", slog.String("error", err.Error()))
		}

	loop:
		for {
			select {
			case <-app.ctx.Done():
				break loop
			case <-time.After(5 * time.Minute):
				if err := app.runCustom(); err != nil {
					slog.Error("error in app.runCustom()", slog.String("error", err.Error()))
				}
			}
		}
	}()

	slog.Info("started git-security")

	return func() error {
		slog.Info("stopping git-security")
		fiberApp.Shutdown()
		cancel()
		wg.Wait()
		slog.Info("stopped git-security")
		return nil
	}, nil
}

func (app *GitSecurityApp) createIndices(ctx context.Context) error {
	for _, idxToCreate := range []string{"id", "is_archived", "owner.login", "primary_language.name"} {
		slog.Info(
			"creating database index if needed",
			slog.String("field", idxToCreate),
		)
		if _, err := app.db.Collection("repositories").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{idxToCreate: 1},
		}); err != nil {
			return err
		}
	}
	for _, idxToCreate := range []string{"key", "order"} {
		if _, err := app.db.Collection("columns").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.M{idxToCreate: 1},
			Options: &options.IndexOptions{Unique: func(b bool) *bool { return &b }(true)},
		}); err != nil {
			return err
		}
	}
	for _, idxToCreate := range []string{"field"} {
		if _, err := app.db.Collection("customs").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.M{idxToCreate: 1},
			Options: &options.IndexOptions{Unique: func(b bool) *bool { return &b }(true)},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (app *GitSecurityApp) createDefaultColumns(ctx context.Context) error {
	defaultColumns := []config.Column{
		{
			Type:           "string",
			Title:          "Organization",
			Description:    "",
			Key:            "owner.login",
			Width:          150,
			Show:           true,
			Filter:         true,
			FilterExpanded: true,
		},
		{
			Type:        "string",
			Title:       "Language",
			Description: "",
			Key:         "primary_language.name",
			Width:       200,
			Show:        true,
			Filter:      true,
		},
		{
			Type:        "string",
			Title:       "Default Branch",
			Description: "The default branch is considered the base branch in your repository, against which all pull requests and code commits are automatically made, unless you specify a different branch.",
			Key:         "default_branch.name",
			Width:       100,
			Show:        true,
			Filter:      true,
		},
		{
			Type:        "string",
			Title:       "Branch Protection Rule",
			Description: "",
			Key:         "default_branch.branch_protection_rule.pattern",
			Width:       150,
			Show:        true,
			Filter:      true,
		},
		{
			Type:        "boolean",
			Title:       "Requires PR?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.requires_approving_reviews",
			Width:       150,
			Show:        true,
			Filter:      true,
		},
		{
			Type:        "boolean",
			Title:       "Signed commits?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.requires_commit_signatures",
			Width:       150,
			Show:        true,
			Filter:      true,
		},
	}

	// check if this is blank columns collection
	cursor, err := app.db.Collection("columns").Find(app.ctx, bson.D{})
	if err != nil {
		return err
	}
	var columns []config.Column
	if err := cursor.All(app.ctx, &columns); err != nil {
		return err
	}
	cursor.Close(app.ctx)
	if len(columns) > 0 {
		return nil
	}

	o, _ := lexorank.Rank("", "")
	for _, c := range defaultColumns {
		c.Order = o
		// TODO: wait for new FerretDB release
		// filter := bson.D{{Key: "key", Value: c.Key}}
		// update := bson.D{{Key: "$setOnInsert", Value: c}}
		// _, err := app.db.Collection("columns").
		// 	UpdateOne(app.ctx, filter, update, options.Update().SetUpsert(true))
		// if err != nil {
		// 	return err
		// }
		if _, err := app.db.Collection("columns").
			InsertOne(app.ctx, c); err != nil {
			slog.Error("error in insert column config")
		}
		o, _ = lexorank.Rank(o, "")
	}
	return nil
}

func (app *GitSecurityApp) fetch() error {
	orgs, err := app.g.GetOrganizations()
	if err != nil {
		return err
	}
	slog.Debug("orgs fetched", slog.Int("count", len(orgs)))

	for _, org := range orgs {
		slog.Debug("fetching org repos", slog.String("org", org.Login))
		repos, err := app.g.GetRepos(org.Login)
		if err != nil {
			return err
		}
		for _, repo := range repos {
			filter := bson.D{{Key: "id", Value: repo.ID}}
			update := bson.D{{Key: "$set", Value: repo}}
			_, err = app.db.Collection("repositories").
				UpdateOne(app.ctx, filter, update, options.Update().SetUpsert(true))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
