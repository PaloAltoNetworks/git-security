package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/db"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

type gitHubMock struct {
	archiveRepositoryErr error
}

func (ghm *gitHubMock) ArchiveRepository(repoID string, archive bool) error {
	return ghm.archiveRepositoryErr
}

func (ghm *gitHubMock) CreateBranchProtectionRule(repoID, pattern string) error {
	return nil
}

func (ghm *gitHubMock) GetOrganizations() ([]*gh.Organization, error) {
	return nil, nil
}

func (ghm *gitHubMock) GetRepos(orgName string) ([]*gh.Repository, error) {
	return nil, nil
}

func (ghm *gitHubMock) UpdateBranchProtectionRule(branchProtectionRuleID, field string, value interface{}) error {
	return nil
}

func (ghm *gitHubMock) GetRepo(orgName, repoName string) (*gh.Repository, error) {
	return &gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID:   "foobar",
			Name: repoName,
		},
	}, nil
}

func (ghm *gitHubMock) UpdatePreceiveHook(orgName string, repoName string, hookName string, enabled bool) error {
	return nil
}

func TestArchiveRepoError(t *testing.T) {
	teardown, db, mdb := db.SetupDBForTest(t)
	defer teardown()

	repo := gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID: "foobar",
		},
	}
	db.UpdateRepository(repo.ID, bson.D{{Key: "$set", Value: repo}}, true)

	a := api{
		ctx: context.Background(),
		db:  mdb,
		dbw: db,
		g: &gitHubMock{
			archiveRepositoryErr: errors.New("archive error"),
		},
	}

	app := fiber.New()
	app.Post("/test", a.ArchiveRepo)

	b, err := json.Marshal(struct {
		IDs         []string    `json:"ids"`
		UpdateValue interface{} `json:"updateValue"`
	}{
		IDs:         []string{"foobar", "2"},
		UpdateValue: true,
	})
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestArchiveRepoNormal(t *testing.T) {
	teardown, db, mdb := db.SetupDBForTest(t)
	defer teardown()

	repo := gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID: "foobar",
		},
	}
	db.UpdateRepository(repo.ID, bson.D{{Key: "$set", Value: repo}}, true)

	a := api{
		ctx: context.Background(),
		db:  mdb,
		dbw: db,
		g:   &gitHubMock{},
	}

	app := fiber.New()
	app.Post("/test", a.ArchiveRepo)

	b, err := json.Marshal(struct {
		IDs         []string    `json:"ids"`
		UpdateValue interface{} `json:"updateValue"`
	}{
		IDs:         []string{"foobar"},
		UpdateValue: true,
	})
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
