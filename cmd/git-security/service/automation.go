package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kballard/go-shellquote"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/security"
)

func (app *GitSecurityApp) runAutomation() error {
	slog.Info(("start runAutomation()"))
	filters := bson.D{
		{
			Key:   "is_archived",
			Value: false,
		},
	}
	repos, err := app.dbw.ReadRepositories(filters)
	if err != nil {
		return err
	}

	for _, repo := range repos {
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

		for _, automation := range automations {
			// prereq check
			if !automation.Enabled || len(automation.Image) == 0 || len(automation.Command) == 0 {
				continue
			}

			if proceedWithRightCondition(repo, automation) {
				// repo struct to json
				b, err := json.Marshal(repo)
				if err != nil {
					slog.Error(
						"error in json.Marshal the repo",
						slog.String("error", err.Error()),
						slog.String("repo", repo.NameWithOwner),
					)
					continue
				}

				// do automation logic
				// add envs + decrypt
				envs := []config.EnvKeyValue{
					{
						Key:   "GIT_REPO_JSON",
						Value: string(b),
					},
				}
				for _, e := range automation.Envs {
					v, err := security.Decrypt(e.Value, app.key)
					if err != nil {
						slog.Error(
							"error in security.Decrypt()",
							slog.String("error", err.Error()),
							slog.String("encrypted", e.Value),
						)
						return err
					}
					envs = append(envs, config.EnvKeyValue{
						Key:   e.Key,
						Value: v,
					})
				}

				if err := app.runSingleAutomation(automation.Image, automation.Command, envs); err != nil {
					slog.Error("error in runSingleAutomation()", slog.String("error", err.Error()))
				}
			}
		}
	}

	return nil
}

func proceedWithRightCondition(repo *gh.Repository, automation config.Automation) bool {
	matchPattern := false
	for _, p := range strings.Split(automation.Pattern, ",") {
		p := strings.Trim(p, " ")
		if len(p) == 0 {
			continue
		}
		if wildcard.Match(p, repo.NameWithOwner) {
			matchPattern = true
			break
		}
	}
	if !matchPattern {
		return false
	}

	for _, e := range strings.Split(automation.Exclude, ",") {
		e := strings.Trim(e, " ")
		if len(e) == 0 {
			continue
		}
		if wildcard.Match(e, repo.NameWithOwner) {
			return false
		}
	}

	// last check on owner
	if strings.Trim(automation.Owner, " ") != "" {
		for _, o := range strings.Split(automation.Owner, ",") {
			o := strings.Trim(o, " ")
			if len(o) == 0 {
				continue
			}
			if wildcard.Match(o, repo.RepoOwner) {
				return true
			}
		}
		return false
	}
	return true
}

func (app *GitSecurityApp) runSingleAutomation(image, command string, envs []config.EnvKeyValue) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("error in NewClientWithOpts()", slog.String("error", err.Error()))
		return err
	}

	c, err := shellquote.Split(command)
	if err != nil {
		slog.Error("error in shellquote.Split()", slog.String("error", err.Error()))
		return err
	}

	e := make([]string, 0)
	masked := make([]string, 0)
	for _, ekv := range envs {
		k := strings.TrimSpace(ekv.Key)
		if k != "" {
			e = append(e, fmt.Sprintf("%s=%s", k, ekv.Value))
			if k != "GIT_REPO_JSON" {
				masked = append(masked,
					fmt.Sprintf("%s=%s", k, strings.Repeat("*", utf8.RuneCountInString(ekv.Value))))
			} else {
				masked = append(masked, fmt.Sprintf("%s=%s", k, ekv.Value))
			}
		}
	}

	slog.Debug("automation: create container",
		slog.String("image", image),
		slog.Any("Cmd", c),
		slog.Any("Env", masked),
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
			slog.Debug("automation: pull image", slog.String("image", image))
			reader, err := cli.ImagePull(app.ctx, image, types.ImagePullOptions{})
			if err != nil {
				slog.Error("error in ImagePull()", slog.String("error", err.Error()))
				return err
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
				return err
			}
		} else {
			slog.Error("error in ContainerCreate()", slog.String("error", err.Error()))
			return err
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
		return err
	}

	statusCh, errCh := cli.ContainerWait(app.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("error in ContainerWait()", slog.String("error", err.Error()))
			return err
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(app.ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		slog.Error("error in ContainerLogs()", slog.String("error", err.Error()))
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(out)

	sc := bufio.NewScanner(buf)
	b := make([]byte, 0, 1024*1024)
	sc.Buffer(b, 102400*1024)
	var line string
	for sc.Scan() {
		line = sc.Text()
		slog.Debug("container output",
			slog.String("output", buf.String()),
			slog.String("line", line),
		)
	}
	if err := sc.Err(); err != nil {
		slog.Error("error in Scan()", slog.String("error", err.Error()))
		return err
	}

	return nil
}
