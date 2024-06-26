package db

import (
	"context"
	"strconv"
	"testing"
	"time"

	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestUpdateRepositorySimple(t *testing.T) {
	teardown, db, _ := SetupDBForTest(t)
	defer teardown()

	repo := gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID: "foobar",
		},
	}

	repos, err := db.ReadRepositories(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 0, len(repos))

	db.UpdateRepository(repo.ID, bson.D{{Key: "$set", Value: repo}}, true)
	repos, err = db.ReadRepositories(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 1, len(repos))
	assert.Equal(t, "foobar", repos[0].ID)

	log, err := db.ReadChangelog(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 1, len(log))
	assert.Equal(t, "New Repo", log[0].Field)
	assert.Equal(t, "foobar", log[0].RepoID)

	repo.IsArchived = true
	r, err := db.UpdateRepository(repo.ID, bson.D{{Key: "$set", Value: repo}}, true) // check return value
	require.Nil(t, err)
	assert.Equal(t, true, r.IsArchived)

	log, err = db.ReadChangelog(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 2, len(log))
	assert.Equal(t, "IsArchived", log[1].Field)
	assert.Equal(t, "false", log[1].From)
	assert.Equal(t, "true", log[1].To)
}

func TestCreateDiffLog(t *testing.T) {
	results := createDiffLog(
		gh.Repository{
			RepoOwner: "123",
			FetchedAt: time.Now(),
			GqlRepository: &gh.GqlRepository{
				ID:         "foobar",
				IsArchived: false,
			},
		},
		gh.Repository{
			RepoOwner: "1234",
			FetchedAt: time.Now().AddDate(0, 0, 1),
			GqlRepository: &gh.GqlRepository{
				ID:         "foobar",
				IsArchived: true,
			},
		})
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "123", results["RepoOwner"][0])
	assert.Equal(t, "1234", results["RepoOwner"][1])
	assert.Equal(t, "false", results["IsArchived"][0])
	assert.Equal(t, "true", results["IsArchived"][1])
}

func TestCreateDiffLogMapAndAddToArray(t *testing.T) {
	results := createDiffLog(
		gh.Repository{
			RepoOwner: "123",
			Customs: map[string]interface{}{
				"pre-receive-hooks": []string{"ggshield"},
			},
		},
		gh.Repository{
			RepoOwner: "123",
			Customs: map[string]interface{}{
				"new-custom":        123,
				"pre-receive-hooks": []string{"ggshield", "newhook"},
			},
		},
	)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "", results["Customs.pre-receive-hooks"][0])
	assert.Equal(t, "newhook", results["Customs.pre-receive-hooks"][1])
	assert.Equal(t, "", results["Customs.new-custom"][0])
	assert.Equal(t, "123", results["Customs.new-custom"][1])
}

func TestCreateDiffLogMapAndAddRemoved(t *testing.T) {
	results := createDiffLog(
		gh.Repository{
			RepoOwner: "123",
			Customs: map[string]interface{}{
				"pre-receive-hooks": []string{"ggshield", "newhook"},
			},
		},
		gh.Repository{
			RepoOwner: "123",
			Customs: map[string]interface{}{
				"pre-receive-hooks": []string{"newhook"},
			},
		},
	)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "ggshield", results["Customs.pre-receive-hooks"][0])
	assert.Equal(t, "", results["Customs.pre-receive-hooks"][1])
}

func TestUpdateRepositoriesByIDs(t *testing.T) {
	teardown, db, _ := SetupDBForTest(t)
	defer teardown()

	for i := range 10 {
		update := bson.D{{Key: "$set", Value: gh.Repository{
			GqlRepository: &gh.GqlRepository{
				ID:         strconv.Itoa(i),
				IsArchived: false,
			},
		}}}
		db.UpdateRepository(strconv.Itoa(i), update, true)
	}
	repos, err := db.ReadRepositories(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 10, len(repos))

	update := bson.D{{Key: "$set", Value: bson.M{"is_archived": true}}}
	updatedRepos, err := db.UpdateRepositoriesByIDs([]string{"0", "1", "2"}, update)
	require.Nil(t, err)
	require.Equal(t, 3, len(updatedRepos))
	for _, r := range updatedRepos {
		assert.Equal(t, true, r.IsArchived)
	}

	repos, err = db.ReadRepositories(bson.D{
		{
			Key:   "is_archived",
			Value: false,
		},
	})
	require.Nil(t, err)
	require.Equal(t, 7, len(repos))

	log, err := db.ReadChangelog(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 13, len(log))
}

func TestUpdateRepositories(t *testing.T) {
	teardown, db, _ := SetupDBForTest(t)
	defer teardown()

	for i := range 100 {
		update := bson.D{{Key: "$set", Value: gh.Repository{
			GqlRepository: &gh.GqlRepository{
				ID:         strconv.Itoa(i),
				IsArchived: false,
			},
		}}}
		db.UpdateRepository(strconv.Itoa(i), update, true)
	}

	filters := bson.D{{Key: "is_archived", Value: false}}
	update := bson.D{{Key: "$set", Value: bson.M{
		"is_archived": true,
		"repo_owner":  "foobar",
	}}}
	updatedRepos, err := db.UpdateRepositories(filters, update)
	require.Nil(t, err)
	require.Equal(t, 100, len(updatedRepos))
	for _, r := range updatedRepos {
		assert.Equal(t, true, r.IsArchived)
		assert.Equal(t, "foobar", r.RepoOwner)
	}

	repos, err := db.ReadRepositories(bson.D{
		{
			Key:   "is_archived",
			Value: true,
		},
	})
	require.Nil(t, err)
	require.Equal(t, 100, len(repos))

	log, err := db.ReadChangelog(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 300, len(log))
}

func TestDeleteRepositories(t *testing.T) {
	teardown, db, _ := SetupDBForTest(t)
	defer teardown()

	for i := range 100 {
		update := bson.D{{Key: "$set", Value: gh.Repository{
			GqlRepository: &gh.GqlRepository{
				ID: strconv.Itoa(i),
			},
			FetchedAt: time.Now().AddDate(0, 0, i*-1),
		}}}
		db.UpdateRepository(strconv.Itoa(i), update, true)
	}

	update := bson.D{{Key: "$set", Value: gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID: "empty fetched_at",
		},
	}}}
	db.UpdateRepository("empty fetched_at", update, true)

	err := db.DeleteRepositories(time.Now().AddDate(0, 0, -30))
	require.Nil(t, err)

	repos, err := db.ReadRepositories(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 30, len(repos))

	log, err := db.ReadChangelog(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 172, len(log))
}

// corner case when the server starts up
// runCustoms() and deleteOldRepos() happens at the same time
func TestBugNewRepoChangelogEmptyName(t *testing.T) {
	teardown, db, mdb := SetupDBForTest(t)
	defer teardown()

	// the record exists before the server starts up
	mdb.Collection(repositoriesTableName).InsertOne(context.Background(), gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID:   "foobar",
			Name: "reponame",
		},
		FetchedAt: time.Now().AddDate(0, 0, -35),
	})

	// in runCustom(), ReadRepositories() runs and loops
	repos, err := db.ReadRepositories(bson.D{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(repos))

	// delete routine happens to run after "repos" was fetched
	err = db.DeleteRepositories(time.Now().AddDate(0, 0, -30))
	require.Nil(t, err)

	// in runCustom(), app.dbw.UpdateRepository(repo.ID, update) runs but the record is gone
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "customs", Value: make(map[string]interface{})},
		{Key: "custom_run_at", Value: time.Now()},
	}}}
	_, err = db.UpdateRepository("foobar", update, false)
	require.Nil(t, err)

	repos, err = db.ReadRepositories(bson.D{})
	require.Nil(t, err)
	assert.Equal(t, 0, len(repos))

	log, err := db.ReadChangelog(bson.D{})
	require.Nil(t, err)
	require.Equal(t, 1, len(log))
}
