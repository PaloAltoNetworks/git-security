package config

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserViewFilter struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	FilterExpanded bool               `bson:"filter_expanded" json:"filter_expanded"`
}

type UserView struct {
	ID       primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	Username string               `bson:"username" json:"username"`
	Filters  []UserViewFilter     `bson:"filters" json:"filters"`
	Columns  []primitive.ObjectID `bson:"columns" json:"columns"`
}
