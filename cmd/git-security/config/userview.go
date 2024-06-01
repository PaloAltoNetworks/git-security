package config

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserViewFilter struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	FilterExpanded bool               `bson:"filter_expanded" json:"filter_expanded"`
}

type UserView struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	Username     string               `bson:"username" json:"username"`
	ShowArchived bool                 `bson:"show_archived" json:"show_archived"`
	Filters      []UserViewFilter     `bson:"filters" json:"filters"`
	Columns      []primitive.ObjectID `bson:"columns" json:"columns"`
}

type Logged struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username string             `bson:"username" json:"username"`
	Start    time.Time          `bson:"start" json:"start"`
	End      time.Time          `bson:"end" json:"end"`
	Duration int                `bson:"duration" json:"duration"`
}
