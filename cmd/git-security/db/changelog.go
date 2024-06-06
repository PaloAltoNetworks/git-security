package db

import (
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
)

const (
	changeLogTableName = "changelog"
)

type ChangeLog struct {
	RepoID        string `bson:"repo_id" json:"repo_id"`
	GitHubHost    string `bson:"github_host" json:"github_host"`
	Name          string `bson:"name" json:"name"`
	NameWithOwner string `bson:"full_name" json:"full_name"`
	Owner         struct {
		Login string `bson:"login" json:"login"`
	} `bson:"owner" json:"owner"`
	RepoOwnerID      primitive.ObjectID `bson:"repo_owner_id,omitempty" json:"repo_owner_id,omitempty"`
	RepoOwner        string             `bson:"repo_owner,omitempty" json:"repo_owner,omitempty"`
	RepoOwnerContact string             `bson:"repo_owner_contact,omitempty" json:"repo_owner_contact,omitempty"`
	Field            string             `bson:"field" json:"field"`
	From             string             `bson:"from" json:"from"`
	To               string             `bson:"to" json:"to"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
}

func (dbi *DatabaseImpl) CreateChangelogIndices() error {
	for _, idxToCreate := range []string{
		"owner.login",
		"name",
		"repo_owner",
		"field",
		"from",
		"to",
	} {
		if _, err := dbi.db.Collection(changeLogTableName).Indexes().CreateOne(dbi.ctx, mongo.IndexModel{
			Keys: bson.M{idxToCreate: 1},
		}); err != nil {
			return err
		}
	}
	for _, idxToCreate := range []string{"created_at"} {
		if _, err := dbi.db.Collection(changeLogTableName).Indexes().CreateOne(dbi.ctx, mongo.IndexModel{
			Keys: bson.M{idxToCreate: -1},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (dbi *DatabaseImpl) ReadChangelog(filters interface{}) ([]*ChangeLog, error) {
	cursor, err := dbi.db.Collection(changeLogTableName).Find(dbi.ctx, filters)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(dbi.ctx)

	log := []*ChangeLog{}
	if err := cursor.All(dbi.ctx, &log); err != nil {
		return nil, err
	}

	return log, nil
}

func (dbi *DatabaseImpl) CreateChangelog(repo *gh.Repository, field, from, to string) error {
	if _, err := dbi.db.Collection(changeLogTableName).InsertOne(
		dbi.ctx,
		ChangeLog{
			RepoID:        repo.ID,
			GitHubHost:    repo.GitHubHost,
			Name:          repo.Name,
			NameWithOwner: repo.NameWithOwner,
			Owner: struct {
				Login string `bson:"login" json:"login"`
			}{
				Login: repo.Owner.Login,
			},
			RepoOwnerID:      repo.RepoOwnerID,
			RepoOwner:        repo.RepoOwner,
			RepoOwnerContact: repo.RepoOwnerContact,
			Field:            field,
			From:             from,
			To:               to,
			CreatedAt:        time.Now(),
		},
	); err != nil {
		slog.Error("error in inserting a changelog entry", slog.String("error", err.Error()))
		return err
	}
	return nil
}
