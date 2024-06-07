package db

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/r3labs/diff/v3"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
)

const (
	repositoriesTableName = "repositories"
)

var (
	ignoredChangelogFields = map[string]struct{}{
		"CustomRunAt":     {},
		"DiskUsage":       {},
		"FetchedAt":       {},
		"LastCommittedAt": {},
		"UpdatedAt":       {},
	}
)

func (dbi *DatabaseImpl) ReadRepositories(filters interface{}) ([]*gh.Repository, error) {
	cursor, err := dbi.db.Collection(repositoriesTableName).Find(dbi.ctx, filters)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(dbi.ctx)

	repos := []*gh.Repository{}
	if err := cursor.All(dbi.ctx, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (dbi *DatabaseImpl) UpdateRepositories(
	filters interface{},
	update interface{},
) ([]*gh.Repository, error) {
	dbi.mu.Lock()
	defer dbi.mu.Unlock()

	// find all the repo with the filters, and create IDs array
	repoIDs := make([]string, 0)
	repos, err := dbi.ReadRepositories(filters)
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		r, ok, err := dbi.getRepoFromCache(repo.ID)
		if err != nil {
			return nil, err
		}
		if ok {
			repoIDs = append(repoIDs, r.ID)
		}
	}

	// update repos based on IDs array (not the filter)
	idFilters := bson.M{"id": bson.M{"$in": repoIDs}}
	if _, err := dbi.db.Collection(repositoriesTableName).UpdateMany(
		dbi.ctx,
		idFilters,
		update,
	); err != nil {
		slog.Error("error in updating the repos", slog.String("error", err.Error()))
		return nil, err
	}

	// find all repos based on the IDs
	updated, err := dbi.ReadRepositories(idFilters)
	if err != nil {
		return nil, err
	}
	for _, repo := range updated {
		r, ok, err := dbi.getRepoFromCache(repo.ID)
		if err != nil {
			return nil, err
		}
		if ok {
			for field, change := range createDiffLog(r, *repo) {
				dbi.CreateChangelog(repo, field, change[0], change[1])
			}
			dbi.repos[repo.ID] = *repo
		}
	}

	return updated, nil
}

func (dbi *DatabaseImpl) UpdateRepositoriesByIDs(
	repoIDs []string,
	update interface{},
) ([]*gh.Repository, error) {
	dbi.mu.Lock()
	defer dbi.mu.Unlock()

	filter := bson.M{"id": bson.M{"$in": repoIDs}}
	if _, err := dbi.db.Collection(repositoriesTableName).UpdateMany(dbi.ctx, filter, update); err != nil {
		slog.Error("error in updating the repos", slog.String("error", err.Error()))
		return nil, err
	}

	repos, err := dbi.ReadRepositories(filter)
	if err != nil {
		return nil, err
	}

	// for each repo, compare the before and after
	for _, repo := range repos {
		r, ok, err := dbi.getRepoFromCache(repo.ID)
		if err != nil {
			return nil, err
		}
		if ok {
			for field, change := range createDiffLog(r, *repo) {
				dbi.CreateChangelog(repo, field, change[0], change[1])
			}
			dbi.repos[repo.ID] = *repo
		}
	}

	return repos, nil
}

func (dbi *DatabaseImpl) getRepoFromCache(repoID string) (gh.Repository, bool, error) {
	r, ok := dbi.repos[repoID]
	if !ok {
		// repo not in cache, get it from database
		if err := dbi.db.Collection(repositoriesTableName).FindOne(
			dbi.ctx,
			bson.D{{Key: "id", Value: repoID}},
		).Decode(&r); err != nil {
			if err != mongo.ErrNoDocuments {
				slog.Error("error in finding the repo", slog.String("err", err.Error()))
				return r, false, err
			}
			// can't find it in database
			return r, false, nil
		} else {
			dbi.repos[repoID] = r
		}
	}
	return r, true, nil
}

func (dbi *DatabaseImpl) UpdateRepository(repoID string, update interface{}) (*gh.Repository, error) {
	dbi.mu.Lock()
	defer dbi.mu.Unlock()

	r, ok, err := dbi.getRepoFromCache(repoID)
	if err != nil {
		return nil, err
	}

	// do the update
	var newRecord *gh.Repository
	filter := bson.D{{Key: "id", Value: repoID}}
	rd := options.After
	options := &options.FindOneAndUpdateOptions{
		Upsert:         new(bool),
		ReturnDocument: &rd,
	}
	*options.Upsert = true

	if err := dbi.db.Collection(repositoriesTableName).FindOneAndUpdate(
		dbi.ctx,
		filter,
		update,
		options,
	).Decode(&newRecord); err != nil {
		if err != mongo.ErrNoDocuments {
			slog.Error(
				"error in FindOneAndUpdate the repo",
				slog.String("id", repoID),
				slog.String("err", err.Error()),
			)
			return nil, err
		}
	}

	if ok {
		// update
		for field, change := range createDiffLog(r, *newRecord) {
			dbi.CreateChangelog(newRecord, field, change[0], change[1])
		}
	} else if newRecord.GqlRepository != nil && newRecord.ID != "" {
		// create
		dbi.CreateChangelog(newRecord, "New Repo", "", "")
	}

	// put the latest version back to cache
	dbi.repos[repoID] = *newRecord

	return newRecord, nil
}

func (dbi *DatabaseImpl) DeleteRepositories(before time.Time) error {
	dbi.mu.Lock()
	defer dbi.mu.Unlock()

	// Create a filter that matches documents where FetchedAt is older than t
	filter := bson.D{{Key: "fetched_at", Value: bson.D{{Key: "$lt", Value: before}}}}

	// find the existing documents first
	repos, err := dbi.ReadRepositories(filter)
	if err != nil {
		return err
	}

	// Delete the documents from the repositories collection
	if _, err := dbi.db.Collection(repositoriesTableName).DeleteMany(dbi.ctx, filter); err != nil {
		return err
	}

	for _, repo := range repos {
		delete(dbi.repos, repo.ID)
		dbi.CreateChangelog(repo, "Delete Repo", "", "")
	}

	return nil
}

func createDiffLog(before, after gh.Repository) map[string][2]string {
	results := make(map[string][2]string)
	changelog, _ := diff.Diff(before, after)
	for _, cl := range changelog {
		if cl.Path[0] == "Customs" {
			field := fmt.Sprintf("Customs.%s", cl.Path[1])
			results[field] = [...]string{cast.ToString(cl.From), cast.ToString(cl.To)}
		} else {
			if cl.Type == diff.UPDATE {
				field := cl.Path[len(cl.Path)-1]
				if _, ok := ignoredChangelogFields[field]; !ok {
					results[field] = [...]string{cast.ToString(cl.From), cast.ToString(cl.To)}
				}
			}
		}
	}
	return results
}
