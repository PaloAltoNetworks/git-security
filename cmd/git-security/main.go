package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/service"
	flag "github.com/eekwong/go-common-flags"
	"github.com/eekwong/go-interruptible-service"
	"github.com/urfave/cli/v2"
)

func main() {

	flags := make([]cli.Flag, 0)
	flag.AddGitHubFlags(&flags)
	flag.AddHttpFlags(&flags)
	flag.AddHttpsFlags(&flags)
	flag.AddPostgresFlags(&flags)
	flag.AddMongoFlags(&flags)
	flag.AddOktaFlags(&flags)

	flags = append(flags, &cli.BoolFlag{
		Name:    "debug",
		Value:   false,
		Usage:   "debug mode",
		EnvVars: []string{"GIT_SECURITY_DEBUG"},
	})

	flags = append(flags, &cli.StringFlag{
		Name:    "key",
		Usage:   "key for encrypting the env variable values in DB",
		EnvVars: []string{"GIT_SECURITY_KEY"},
	})

	flags = append(flags, &cli.StringFlag{
		Name:    "cacert",
		Usage:   "cacert for accessing the GitHub",
		EnvVars: []string{"GIT_SECURITY_CACERT"},
	})

	flags = append(flags, &cli.StringSliceFlag{
		Name:    "admin-usernames",
		Usage:   "basic auth admin username",
		Value:   cli.NewStringSlice("admin"),
		EnvVars: []string{"GIT_SECURITY_ADMIN_USERNAMES"},
	})

	flags = append(flags, &cli.StringSliceFlag{
		Name:    "admin-passwords",
		Usage:   "basic auth admin password",
		Value:   cli.NewStringSlice("changeme"),
		EnvVars: []string{"GIT_SECURITY_ADMIN_PASSWORDS"},
	})

	flags = append(flags, &cli.StringFlag{
		Name:    "db",
		Usage:   "Sqlite (sqlite), PostgreSQL (pg) or Mongo (mongo) as database backend",
		Value:   "sqlite",
		EnvVars: []string{"GIT_SECURITY_DB"},
	})

	app := &cli.App{
		Name:    "github-security",
		Version: "v0.1.0",
		Flags:   flags,
		Action: func(c *cli.Context) error {
			var opts *slog.HandlerOptions
			if c.Bool("debug") {
				opts = &slog.HandlerOptions{
					Level: slog.LevelDebug,
				}
			}
			logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
			slog.SetDefault(logger)

			return interruptible.Run(service.New(
				&service.Opts{
					GitHub:         flag.GetGitHubOpts(c),
					Http:           flag.GetHttpOpts(c),
					Https:          flag.GetHttpsOpts(c),
					Postgres:       flag.GetPostgresOpts(c),
					Mongo:          flag.GetMongoOpts(c),
					Okta:           flag.GetOktaOpts(c),
					Key:            c.String("key"),
					CACert:         c.String("cacert"),
					DB:             c.String("db"),
					AdminUsernames: c.StringSlice("admin-usernames"),
					AdminPasswords: c.StringSlice("admin-passwords"),
				},
			))
		},
		Commands: []*cli.Command{
			{
				Name:  "generate-key",
				Usage: "generate a random encryption key for GIT_SECURITY_KEY",
				Action: func(c *cli.Context) error {
					key := make([]byte, 32)
					_, err := rand.Read(key)
					if err != nil {
						return nil
					}
					fmt.Println(base64.StdEncoding.EncodeToString(key))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("error in app.Run()", slog.String("error", err.Error()))
	}
}
