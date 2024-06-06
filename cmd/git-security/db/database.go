package db

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
)

type Database interface {
	CreateChangelog(repo *gh.Repository, field, from, to string) error
	CreateChangelogIndices() error
	DeleteRepositories(before time.Time) error
	ReadChangelog(filters interface{}) ([]*ChangeLog, error)
	ReadRepositories(filters interface{}) ([]*gh.Repository, error)
	UpdateRepositories(filters interface{}, update interface{}) ([]*gh.Repository, error)
	UpdateRepositoriesByIDs(repoIDs []string, update interface{}) ([]*gh.Repository, error)
	UpdateRepository(repoID string, update interface{}) (*gh.Repository, error)
}

type DatabaseImpl struct {
	ctx   context.Context
	db    *mongo.Database
	repos map[string]gh.Repository
	mu    sync.Mutex
}

func New(ctx context.Context, db *mongo.Database) Database {
	return &DatabaseImpl{
		ctx:   ctx,
		db:    db,
		repos: make(map[string]gh.Repository),
	}
}
