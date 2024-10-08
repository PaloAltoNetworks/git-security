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

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/api"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/db"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
)

const (
	deleteOldDataInterval = 24 * time.Hour
	oldRepos              = -7
)

type Opts struct {
	GitHub            *flag.GitHubOpts
	Http              *flag.HttpOpts
	Https             *flag.HttpsOpts
	Postgres          *flag.PostgresOpts
	Mongo             *flag.MongoOpts
	Okta              *flag.OktaOpts
	Key               string
	CACert            string
	DB                string
	AdminUsernames    []string
	AdminPasswords    []string
	IgnoredCommitters []string
}

type GitSecurityApp struct {
	interruptible.Service
	ctx  context.Context
	opts *Opts
	db   *mongo.Database
	dbw  db.Database
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
			Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
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
	app.dbw = db.New(app.ctx, app.db)

	// create indices
	if err := app.createIndices(ctx); err != nil {
		cancel()
		return nil, err
	}

	// create default columns
	app.createDefaultColumns()

	// setup github clients
	app.g, err = gh.New(ctx, app.opts.GitHub.Host, app.opts.GitHub.PAT, app.opts.CACert, app.opts.IgnoredCommitters)
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
	fiberApp := api.NewFiberApp(
		ctx, app.db, app.dbw, app.g, app.key, app.opts.AdminUsernames, app.opts.AdminPasswords, app.opts.Okta)
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

	// run automation logics
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := app.runAutomation(); err != nil {
			slog.Error("error in app.runAutomation()", slog.String("error", err.Error()))
		}

	loop:
		for {
			select {
			case <-app.ctx.Done():
				break loop
			case <-time.After(5 * time.Minute):
				if err := app.runAutomation(); err != nil {
					slog.Error("error in app.runAutomation()", slog.String("error", err.Error()))
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := app.deleteOldRepos(oldRepos); err != nil {
			slog.Error("error in app.runCustom()", slog.String("error", err.Error()))
		}

	loop:
		for {
			select {
			case <-app.ctx.Done():
				break loop
			case <-time.After(deleteOldDataInterval):
				if err := app.deleteOldRepos(oldRepos); err != nil {
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
	for _, idxToCreate := range []string{"name"} {
		if _, err := app.db.Collection("owners").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.M{idxToCreate: 1},
			Options: &options.IndexOptions{Unique: func(b bool) *bool { return &b }(true)},
		}); err != nil {
			return err
		}
	}
	for _, idxToCreate := range []string{"username", "start", "end"} {
		if _, err := app.db.Collection("logged").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{idxToCreate: -1},
		}); err != nil {
			return err
		}
	}

	if err := app.dbw.CreateChangelogIndices(); err != nil {
		return err
	}

	return nil
}

func (app *GitSecurityApp) createDefaultColumns() error {
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
			Type:        "integer",
			Title:       "Approving Review Count?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.required_approving_review_count",
			Width:       150,
			Show:        true,
			Filter:      true,
		},
		{
			Type:        "boolean",
			Title:       "Dismiss Stale Reviews?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.dismisses_stale_reviews",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Requires Code Owner Reviews?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.requires_code_owner_reviews",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Conversation Resolution?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.requires_conversation_resolution",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Admin Enforced?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.is_admin_enforced",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Allow Force Pushes",
			Description: "",
			Key:         "default_branch.branch_protection_rule.allows_force_pushes",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Allow Deletions?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.allows_deletion",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Signed commits?",
			Description: "",
			Key:         "default_branch.branch_protection_rule.requires_commit_signatures",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "string",
			Title:       "Repo Owner",
			Description: "",
			Key:         "repo_owner",
			Width:       150,
			Show:        true,
			Filter:      true,
		},
		{
			Type:        "reposcore",
			Title:       "Score",
			Description: "",
			Key:         "score",
			Width:       70,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "boolean",
			Title:       "Archived?",
			Description: "",
			Key:         "is_archived",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "string",
			Title:       "Repo Owner Contact",
			Description: "",
			Key:         "repo_owner_contact",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
		{
			Type:        "integer",
			Title:       "Automations Count",
			Description: "",
			Key:         "automations_count",
			Width:       150,
			Show:        false,
			Filter:      false,
		},
	}

	var col config.Column
	if err := app.db.Collection("columns").FindOne(
		app.ctx,
		bson.D{},
		options.FindOne().SetSort(bson.D{{Key: "order", Value: -1}}),
	).Decode(&col); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}

	o, _ := lexorank.Rank(col.Order, "")
	for _, column := range defaultColumns {
		column.Order = o

		// Check if the column exists
		err := app.db.Collection("columns").FindOne(app.ctx, bson.D{{Key: "key", Value: column.Key}}).Err()
		if err != nil {
			// ErrNoDocuments means that the filter did not match any documents in the collection
			if err == mongo.ErrNoDocuments {
				// If the column doesn't exist, insert it
				_, err := app.db.Collection("columns").InsertOne(app.ctx, column)
				if err != nil {
					slog.Error("error in default column InsertOne", slog.String("error", err.Error()))
					return err
				}
				o, _ = lexorank.Rank(o, "")
			} else {
				// If there's an error other than the column not existing, return the error
				return err
			}
		}
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

		// get score and colors
		gs := config.GlobalSettings{
			ScoreColors:  make([]config.ScoreColor, 0),
			ScoreWeights: make([]config.ScoreWeight, 0),
		}
		if err := app.db.Collection("globalSettings").FindOne(
			app.ctx,
			bson.D{},
		).Decode(&gs); err != nil {
			if err != mongo.ErrNoDocuments {
				return err
			}
		}

		// get automations
		cursorAutomation, err := app.db.Collection("automations").Find(app.ctx, bson.D{})
		if err != nil {
			return err
		}
		var automations []config.Automation
		if err := cursorAutomation.All(app.ctx, &automations); err != nil {
			cursorAutomation.Close(app.ctx)
			return err
		}
		cursorAutomation.Close(app.ctx)

		for _, repo := range repos {
			// update score and color
			if err := repo.UpdateRepoScoreAndColor(&gs); err != nil {
				continue
			}

			// update the automation count
			automationsCount := 0
			for _, automation := range automations {
				if proceedWithRightCondition(repo, automation) &&
					automation.Enabled &&
					len(automation.Image) > 0 &&
					len(automation.Command) > 0 {
					automationsCount += 1
				}
			}
			repo.AutomationsCount = automationsCount

			update := bson.D{{Key: "$set", Value: *repo}}
			if _, err := app.dbw.UpdateRepository(repo.ID, update, true); err != nil {
				return err
			}
		}
	}

	return nil
}

func (app *GitSecurityApp) deleteOldRepos(oldRepos int) error {
	return app.dbw.DeleteRepositories(time.Now().AddDate(0, 0, oldRepos))
}
