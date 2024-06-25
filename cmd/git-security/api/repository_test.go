package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	"github.com/PaloAltoNetworks/git-security/cmd/git-security/db"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func TestGetRepositoriesCSVs(t *testing.T) {
	teardown, db, mdb := db.SetupDBForTest(t)
	defer teardown()

	// Insert repository data
	repo := gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID:            "foobar",
			Name:          "repo1",
			NameWithOwner: "owner1/repo1",
			IsDisabled:    true,
		},
	}
	db.UpdateRepository(repo.ID, bson.D{{Key: "$set", Value: repo}}, true)

	// Setup goFiber app for /test route to GetRepositories()
	app := fiber.New()
	a := api{
		ctx: context.Background(),
		db:  mdb,
		dbw: db,
		g:   &gitHubMock{},
		getUsernameFromSession: func(c *fiber.Ctx) (string, error) {
			return "testuser", nil
		},
	}
	app.Post("/test", func(c *fiber.Ctx) error {
		return a.GetRepositories(c)
	})

	// Create columns
	columnID1 := primitive.NewObjectID()
	columnID2 := primitive.NewObjectID()
	columns := []interface{}{
		config.Column{
			ID:          columnID1,
			Type:        "string",
			Title:       "Disabled",
			Description: "is_disabled",
			Key:         "name",
			CSV:         true,
		},
		config.Column{
			ID:          columnID2,
			Type:        "string",
			Title:       "Name with Owner",
			Description: "The repo owner",
			Key:         "full_name",
			CSV:         true,
		},
	}
	_, err := a.db.Collection("columns").InsertMany(a.ctx, columns)
	require.Nil(t, err)

	// Create a userview with column IDs
	uv := config.UserView{
		Columns:  []primitive.ObjectID{columnID1, columnID2},
		Username: "testuser",
	}
	_, err = a.db.Collection("userviews").InsertOne(a.ctx, uv)
	require.Nil(t, err)

	// Setup a filter for POST to /test
	filters := []Filter{
		{
			Type:            "type1",
			Field:           "name",
			Values:          []interface{}{"repo1"},
			Negate:          false,
			IncludeZeroTime: true,
		},
	}
	b, err := json.Marshal(struct {
		Filters []Filter `json:"filters"`
	}{
		Filters: filters,
	})
	require.NoError(t, err, "Failed to marshal request data")

	req := httptest.NewRequest("POST", "/test?csv=true", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.Nil(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))
	assert.Equal(t, "attachment; filename=repos.csv", resp.Header.Get("Content-Disposition"))

	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)

	// Check the CSV data
	csvReader := csv.NewReader(bytes.NewBuffer(body))
	records, err := csvReader.ReadAll()
	require.Nil(t, err) // This particular part was giving error for the FOR loop of columns

	// Check the number of records
	assert.Equal(t, 2, len(records))

	// Check the header record
	expectedHeaders := []string{"Repo Name", "Disabled", "Name with Owner"}
	assert.Equal(t, expectedHeaders, records[0])

	// Check the data record
	expectedData := []string{"repo1", "repo1", "owner1/repo1"}
	assert.Equal(t, expectedData, records[1])
}
