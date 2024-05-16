package config

import "go.mongodb.org/mongo-driver/bson/primitive"

type Owner struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name    string             `bson:"name" json:"name"`
	Contact string             `bson:"contact" json:"contact"`
	Notes   string             `bson:"notes" json:"notes"`
}
