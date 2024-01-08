package service

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/eekwong/git-security/cmd/git-security/config"
	"github.com/eekwong/git-security/cmd/git-security/security"
)

func (app *GitSecurityApp) runCustom() error {
	slog.Info(("start runCustom()"))
	filters := bson.D{
		{
			Key:   "is_archived",
			Value: false,
		},
	}
	opts := options.Find().SetProjection(
		bson.D{
			{Key: "id", Value: 1},
			{Key: "full_name", Value: 1},
			{Key: "customs", Value: 1},
		},
	)
	cursor, err := app.db.Collection("repositories").Find(app.ctx, filters, opts)
	if err != nil {
		return err
	}
	var repos []struct {
		ID            string                 `bson:"id" json:"id"`
		NameWithOwner string                 `bson:"full_name" json:"full_name"`
		Customs       map[string]interface{} `bson:"customs" json:"customs"`
	}
	if err := cursor.All(app.ctx, &repos); err != nil {
		cursor.Close(app.ctx)
		return err
	}
	cursor.Close(app.ctx)

	for _, repo := range repos {
		cursorCustom, err := app.db.Collection("customs").Find(app.ctx, bson.D{})
		if err != nil {
			return err
		}
		var customs []config.Custom
		if err := cursorCustom.All(app.ctx, &customs); err != nil {
			cursorCustom.Close(app.ctx)
			return err
		}
		cursorCustom.Close(app.ctx)

		for _, custom := range customs {
			for _, p := range strings.Split(custom.Pattern, ",") {
				// prereq check
				if len(p) == 0 || len(custom.Image) == 0 || len(custom.Command) == 0 || len(custom.Field) == 0 {
					break
				}
				customRun := false
				var result interface{}
				if wildcard.Match(p, repo.NameWithOwner) {
					// do custom logic
					envs := custom.Envs
					if envs == nil {
						envs = make([]config.EnvKeyValue, 0)
					}

					// decrypt
					for idx := range customs {
						for idy := range customs[idx].Envs {
							customs[idx].Envs[idy].Value, err = security.Decrypt(customs[idx].Envs[idy].Value, app.key)
							if err != nil {
								return err
							}
						}
					}

					envs = append(envs, config.EnvKeyValue{
						Key:   "GIT_REPO",
						Value: repo.NameWithOwner,
					})
					result, err = app.runSingleCustom(custom.Image, custom.Command, envs)
					if err != nil {
						slog.Error("error in runSingleCustom()", slog.String("error", err.Error()))
						result = custom.ErrorValue
					}
					customRun = true
				} else {
					result = custom.DefaultValue
				}

				hasUpdate := false
				if repo.Customs == nil {
					repo.Customs = make(map[string]interface{})
				}
				switch custom.ValueType {
				case "string":
					r := cast.ToString(result)
					if v, ok := repo.Customs[custom.Field]; !ok || cast.ToString(v) != r {
						hasUpdate = true
						repo.Customs[custom.Field] = r
					}
				case "number":
					r := cast.ToFloat64(result)
					if v, ok := repo.Customs[custom.Field]; !ok || cast.ToFloat64(v) != r {
						hasUpdate = true
						repo.Customs[custom.Field] = r
					}
					repo.Customs[custom.Field] = cast.ToFloat64(result)
				case "boolean":
					r := cast.ToBool(result)
					if v, ok := repo.Customs[custom.Field]; !ok || cast.ToBool(v) != r {
						hasUpdate = true
						repo.Customs[custom.Field] = r
					}
					repo.Customs[custom.Field] = cast.ToBool(result)
				}

				if hasUpdate || customRun {
					update := bson.D{{Key: "$set", Value: bson.D{
						{Key: "customs", Value: repo.Customs},
					}}}
					if customRun {
						update = bson.D{{Key: "$set", Value: bson.D{
							{Key: "customs", Value: repo.Customs},
							{Key: "custom_run_at", Value: time.Now()},
						}}}
					}
					filter := bson.D{{Key: "id", Value: repo.ID}}
					_, err = app.db.Collection("repositories").UpdateOne(app.ctx, filter, update)
					if err != nil {
						slog.Error("error in Update()", slog.String("error", err.Error()))
						break
					}
				}
			}
		}
	}

	return nil
}

func (app *GitSecurityApp) runSingleCustom(image, command string, envs []config.EnvKeyValue) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("error in NewClientWithOpts()", slog.String("error", err.Error()))
		return "", err
	}

	c, err := shellquote.Split(command)
	if err != nil {
		slog.Error("error in shellquote.Split()", slog.String("error", err.Error()))
		return "", err
	}

	e := make([]string, 0)
	for _, ekv := range envs {
		k := strings.TrimSpace(ekv.Key)
		if k != "" {
			e = append(e, fmt.Sprintf("%s=%s", k, ekv.Value))
		}
	}

	slog.Debug("custom: create container",
		slog.String("image", image),
		slog.Any("Cmd", c),
		slog.Any("Env", e),
	)
	resp, err := cli.ContainerCreate(app.ctx, &container.Config{
		Image: image,
		Cmd:   c,
		Tty:   true,
		Env:   e,
	}, nil, nil, nil, "")
	if err != nil {
		if strings.Contains(err.Error(), "No such image") {
			// pull the image
			slog.Debug("custom: pull image", slog.String("image", image))
			reader, err := cli.ImagePull(app.ctx, image, types.ImagePullOptions{})
			if err != nil {
				slog.Error("error in ImagePull()", slog.String("error", err.Error()))
				return "", err
			}
			defer reader.Close()
			io.Copy(io.Discard, reader)

			resp, err = cli.ContainerCreate(app.ctx, &container.Config{
				Image: image,
				Cmd:   c,
				Tty:   true,
				Env:   e,
			}, nil, nil, nil, "")
			if err != nil {
				slog.Error("error in ContainerCreate()", slog.String("error", err.Error()))
				return "", err
			}
		} else {
			slog.Error("error in ContainerCreate()", slog.String("error", err.Error()))
			return "", err
		}
	}
	defer func() {
		if err := cli.ContainerRemove(app.ctx, resp.ID, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         true,
		}); err != nil {
			slog.Error("error in ContainerRemove()", slog.String("ID", resp.ID), slog.String("error", err.Error()))
		}
		cli.Close()
	}()

	if err := cli.ContainerStart(app.ctx, resp.ID, container.StartOptions{}); err != nil {
		slog.Error("error in ContainerStart()", slog.String("error", err.Error()))
		return "", err
	}

	statusCh, errCh := cli.ContainerWait(app.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("error in ContainerWait()", slog.String("error", err.Error()))
			return "", err
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(app.ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		slog.Error("error in ContainerLogs()", slog.String("error", err.Error()))
		return "", err
	}

	sc := bufio.NewScanner(out)
	var line string
	for sc.Scan() {
		line = sc.Text()
	}
	if err := sc.Err(); err != nil {
		slog.Error("error in Scan()", slog.String("error", err.Error()))
		return "", err
	}
	return line, nil
}