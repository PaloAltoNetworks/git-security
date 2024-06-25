package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/syncmap"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/db"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	flag "github.com/eekwong/go-common-flags"
)

const (
	modelConf = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, "admin") || g(r.sub, p.sub) && globMatch(r.obj, p.obj) && r.act == p.act
`
)

var newPolicies = [][]string{
	{"user", "/api/v1/repos", "POST"},
	{"user", "/api/v1/repos/*", "POST"},
	{"user", "/api/v1/columns", "GET"},
	{"user", "/api/v1/owners", "GET"},
	{"user", "/api/v1/userview", "GET"},
	{"user", "/api/v1/userview", "PUT"},
	{"user", "/ws", "GET"},
	{"owneradmin", "/api/v1/repos/action/repo-owner", "POST"},
	{"owneradmin", "/api/v1/repos/action/delete-owner/*", "POST"},
	{"owneradmin", "/api/v1/owners", "POST"},
	{"owneradmin", "/api/v1/owner/*", "DELETE"},
	{"owneradmin", "/api/v1/owner/*", "PUT"},
}

var rolesDefined = map[string]struct{}{
	"admin":      {},
	"user":       {},
	"owneradmin": {},
}

type api struct {
	ctx                    context.Context
	db                     *mongo.Database
	dbw                    db.Database
	g                      gh.GitHub
	key                    []byte
	clients                syncmap.Map
	store                  *session.Store
	oktaOpts               *flag.OktaOpts
	enforcer               *casbin.Enforcer
	loggedCache            *expirable.LRU[string, time.Time]
	mu                     sync.Mutex
	getUsernameFromSession func(c *fiber.Ctx) (string, error)
}

func NewFiberApp(
	ctx context.Context,
	db *mongo.Database,
	dbw db.Database,
	g gh.GitHub,
	key []byte,
	adminUsernames []string,
	adminPasswords []string,
	oktaOpts *flag.OktaOpts,
) *fiber.App {
	app := fiber.New()
	app.Use(compress.New())

	app.Use(cors.New())

	store := session.New(session.Config{
		Expiration:   time.Hour,
		KeyLookup:    "cookie:session_id",
		KeyGenerator: utils.UUID,
	})

	a := api{
		ctx:         ctx,
		db:          db,
		dbw:         dbw,
		g:           g,
		key:         key,
		clients:     syncmap.Map{},
		store:       store,
		oktaOpts:    oktaOpts,
		loggedCache: expirable.NewLRU[string, time.Time](1000, nil, time.Hour),
		getUsernameFromSession: func(c *fiber.Ctx) (string, error) {
			sess, err := store.Get(c)
			if err != nil || sess.Get("username") == nil || cast.ToString(sess.Get("username")) == "" {
				return "", c.SendStatus(fiber.StatusForbidden)
			}
			username := cast.ToString(sess.Get("username"))
			return username, nil
		},
	}

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	currentDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	filesDir := filepath.Join(currentDir, "ui")

	// casbin
	a.settingUpCasbinEnforcer()

	if oktaOpts.IsEnabled() {
		app.Get("/login", a.oktaLogin)
		app.Get("/login/callback", a.oktaCallback)
		app.Get("/logout", a.oktaLogout)
		app.Use(a.oktaAuthenticator())

		app.Static("/", filesDir)

		for _, username := range adminUsernames {
			slog.Info("adding admin", slog.String("username", username))
			if _, err := a.enforcer.AddRoleForUser(username, "admin"); err != nil {
				slog.Error("error in adding admins", slog.String("err", err.Error()))
				panic(err)
			}
		}
		a.enforcer.SavePolicy()

		app.Use(func(c *fiber.Ctx) error {
			sess, err := store.Get(c)
			if err != nil || sess.Get("username") == nil || cast.ToString(sess.Get("username")) == "" {
				return c.SendStatus(fiber.StatusForbidden)
			}
			r, err := a.enforcer.Enforce(
				cast.ToString(sess.Get("username")),
				string(c.Request().URI().Path()),
				string(c.Request().Header.Method()),
			)
			if err != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			if !r {
				return c.SendStatus(fiber.StatusForbidden)
			}
			return c.Next()
		})
	} else {
		// check both slice length is the same
		if len(adminUsernames) != len(adminPasswords) {
			panic("admin usernames and passwords should have the same size")
		}

		users := make(map[string]string)
		for idx, username := range adminUsernames {
			users[username] = adminPasswords[idx]
		}
		app.Use(basicauth.New(basicauth.Config{
			Users: users,
		}))
		app.Use(func(c *fiber.Ctx) error {
			sess, err := store.Get(c)
			if err != nil {
				return c.SendStatus(fiber.StatusForbidden)
			}
			sess.Set("username", c.Locals("username"))
			sess.Save()
			return c.Next()
		})

		app.Static("/", filesDir)
	}

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		slog.Info("WebSocket connection established")
		a.clients.Store(c, true)
		defer func() {
			a.clients.Delete(c)
			c.Close()
		}()
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Error("error in socket ReadMessage", slog.String("error", err.Error()))
				}
				break
			}
		}
	}))

	apiRoute := app.Group("/api")
	v1 := apiRoute.Group("/v1")

	// logged time for activity
	v1.Use(a.loggedTime)

	v1.Delete("/column/:id", a.DeleteColumn)
	v1.Delete("/custom/:id", a.DeleteCustom)
	v1.Delete("/owner/:id", a.DeleteOwner)
	v1.Get("/columns", a.GetColumns)
	v1.Get("/customs", a.GetCustoms)
	v1.Get("/globalsettings", a.GetGlobalSettings)
	v1.Get("/logged", a.GetLoggeds)
	v1.Get("/owners", a.GetOwners)
	v1.Get("/roles", a.GetRoles)
	v1.Get("/users", a.GetUsers)
	v1.Get("/userview", a.GetUserView)
	v1.Post("/changelog", a.GetChangelog)
	v1.Post("/changelog/:groupBy", a.GetChangelogGroupBy)
	v1.Post("/columns", a.CreateColumn)
	v1.Post("/customs", a.CreateCustom)
	v1.Post("/columns/order", a.ChangeColumnsOrder)
	v1.Post("/owners", a.CreateOwner)
	v1.Post("/repos", a.GetRepositories)
	v1.Post("/repos/:groupBy", a.GetRepositoriesGroupBy)
	v1.Post("/repos/action/add-branch-protection-rule", a.AddBranchProtectionRule)
	v1.Post("/repos/action/admin-enforced", a.IsAdminEnforced)
	v1.Post("/repos/action/allows-deletions", a.AllowsDeletions)
	v1.Post("/repos/action/allows-force-pushes", a.AllowsForcePushes)
	v1.Post("/repos/action/archive-repo", a.ArchiveRepo)
	v1.Post("/repos/action/delete-owner", a.DeleteRepoOwner)
	v1.Post("/repos/action/dismisses-stale-reviews", a.DismissesStaleReviews)
	v1.Post("/repos/action/pre-receive-hook", a.PreReceiveHook)
	v1.Post("/repos/action/required-approving-review-count", a.RequiredApprovingReviewCount)
	v1.Post("/repos/action/requires-code-owner-reviews", a.RequiresCodeOwnerReviews)
	v1.Post("/repos/action/requires-commit-signatures", a.RequiresCommitSignatures)
	v1.Post("/repos/action/requires-conversation-resolution", a.RequiresConversationResolution)
	v1.Post("/repos/action/requires-status-checks", a.RequiresStatusChecks)
	v1.Post("/repos/action/requires-strict-status-checks", a.RequiresStrictStatusChecks)
	v1.Post("/repos/action/repo-owner", a.AddRepoOwner)
	v1.Post("/repos/action/requires-pr", a.RequiresPR)
	v1.Put("/column/:id", a.UpdateColumn)
	v1.Put("/custom/:id", a.UpdateCustom)
	v1.Put("/globalsettings", a.UpdateGlobalSettings)
	v1.Put("/owner/:id", a.UpdateOwner)
	v1.Put("/user/:name", a.UpdateUserRoles)
	v1.Put("/userview", a.UpdateUserView)

	return app
}

func (a *api) settingUpCasbinEnforcer() {
	// clear all policies
	// RemovePolicies doesn't work with wildcard, we saw left over rules
	if _, err := a.db.Collection("casbin_rule").DeleteMany(a.ctx, bson.D{{Key: "ptype", Value: "p"}}); err != nil {
		slog.Error("error in deleting the policies", slog.String("error", err.Error()))
		panic(err)
	}

	adapter, err := mongodbadapter.NewAdapterByDB(a.db.Client(), &mongodbadapter.AdapterConfig{
		DatabaseName:   a.db.Name(),
		CollectionName: "casbin_rule",
	})
	if err != nil {
		slog.Error("error in creating mongodb casbin adapter", slog.String("err", err.Error()))
		panic(err)
	}
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		slog.Error("error in creating casbin model from string", slog.String("err", err.Error()))
		panic(err)
	}
	a.enforcer, err = casbin.NewEnforcer(m, adapter)
	if err != nil {
		slog.Error("error in creating Enforcer", slog.String("err", err.Error()))
		panic(err)
	}
	a.enforcer.LoadPolicy()
	if _, err := a.enforcer.AddPolicies(newPolicies); err != nil {
		slog.Error("error in adding policies", slog.String("err", err.Error()))
		panic(err)
	}
}

func (a *api) broadcastMessage(repo gh.Repository) {
	// Convert the repo object to a JSON string
	repoJson, err := json.Marshal(repo)
	if err != nil {
		slog.Error(
			"error in broadcastMessage",
			slog.String("error", err.Error()),
			slog.String("repo", repo.Name),
		)
		return
	}

	// Broadcast the JSON string to all connected WebSocket clients
	a.clients.Range(func(k, v interface{}) bool {
		if client, ok := k.(*websocket.Conn); ok {
			if err := client.WriteMessage(websocket.TextMessage, repoJson); err != nil {
				slog.Error(
					"error in socket WriteMessage",
					slog.String("error", err.Error()),
					slog.String("repo", repo.Name),
				)
				client.Close()
				a.clients.Delete(client)
			}
		}
		return true
	})
}

func (a *api) loggedTime(c *fiber.Ctx) error {
	username, err := a.getUsernameFromSession(c)
	if err != nil {
		return err
	}

	a.mu.Lock()
	if ts, ok := a.loggedCache.Get(username); ok && time.Now().Before(ts.Add(10*time.Second)) {
		a.mu.Unlock()
		return c.Next()
	}
	a.loggedCache.Add(username, time.Now())
	a.mu.Unlock()

	// try updating the database in a background go routine
	go func(username string) {
		now := time.Now()
		filter := bson.D{
			{Key: "username", Value: username},
			{Key: "start", Value: bson.M{"$lte": now}},
			{Key: "end", Value: bson.M{"$gte": now}},
		}
		newEnd := time.Now().Add(time.Minute)
		var logged config.Logged
		if err := a.db.Collection("logged").FindOne(
			a.ctx,
			filter,
			options.FindOne().SetSort(bson.D{{Key: "start", Value: -1}}),
		).Decode(&logged); err != nil {
			if err != mongo.ErrNoDocuments {
				slog.Error("error in finding the logged entry", slog.String("err", err.Error()))
				return
			}
			// can't find it, create new record
			if _, err := a.db.Collection("logged").InsertOne(
				a.ctx,
				config.Logged{
					Username: username,
					Start:    now,
					End:      newEnd,
					Duration: int(newEnd.Sub(now).Seconds()),
				},
			); err != nil {
				slog.Error("error in inserting a logged entry", slog.String("error", err.Error()))
				return
			}
			return
		}
		// got the existing record, update the duration and end
		update := bson.D{{Key: "$set", Value: bson.M{
			"end":      newEnd,
			"duration": int(newEnd.Sub(logged.Start).Seconds()),
		}}}
		if _, err := a.db.Collection("logged").UpdateByID(a.ctx, logged.ID, update); err != nil {
			slog.Error("error in updating the logged entry", slog.String("err", err.Error()))
			return
		}
	}(username)

	return c.Next()
}
