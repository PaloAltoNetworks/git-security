package db

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/ferretdb"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	"github.com/stretchr/testify/require"
)

type Database interface {
	CreateChangelog(repo *gh.Repository, field, from, to string) error
	CreateChangelogIndices() error
	DeleteRepositories(before time.Time) error
	ReadChangelog(filters interface{}) ([]*ChangeLog, error)
	ReadRepositories(filters interface{}) ([]*gh.Repository, error)
	UpdateRepositories(filters interface{}, update interface{}) ([]*gh.Repository, error)
	UpdateRepositoriesByIDs(repoIDs []string, update interface{}) ([]*gh.Repository, error)
	UpdateRepository(repoID string, update interface{}, upsert bool) (*gh.Repository, error)
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

func SetupDBForTest(t *testing.T) (func(), Database, *mongo.Database) {
	dir, _ := os.MkdirTemp(os.TempDir(), "")

	listener, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.Nil(t, listener.Close())

	f, err := ferretdb.New(&ferretdb.Config{
		Listener: ferretdb.ListenerConfig{
			TCP: fmt.Sprintf("localhost:%d", port),
		},
		Handler:   "sqlite",
		SQLiteURL: fmt.Sprintf("file:%s/", dir),
	})
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := f.Run(ctx); err != nil {
			require.Nil(t, err)
		}
	}()

	uri := f.MongoDBURI()

	m, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.Nil(t, err)

	return func() {
		cancel()
		wg.Wait()
		os.RemoveAll(dir)
	}, New(ctx, m.Database("public")), m.Database("public")
}
