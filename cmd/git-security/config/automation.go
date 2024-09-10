package config

import "go.mongodb.org/mongo-driver/bson/primitive"

type Automation struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Pattern string             `bson:"pattern" json:"pattern"`
	Owner   string             `bson:"owner" json:"owner"`
	Exclude string             `bson:"exclude" json:"exclude"`
	Image   string             `bson:"image" json:"image"`
	Command string             `bson:"command" json:"command"`
	Envs    []EnvKeyValue      `bson:"envs" json:"envs"`
	Enabled bool               `bson:"enabled" json:"enabled"`
}
