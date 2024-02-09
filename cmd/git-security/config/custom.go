package config

import "go.mongodb.org/mongo-driver/bson/primitive"

type EnvKeyValue struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value" json:"value"`
}

type Custom struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Pattern      string             `bson:"pattern" json:"pattern"`
	Image        string             `bson:"image" json:"image"`
	Command      string             `bson:"command" json:"command"`
	Envs         []EnvKeyValue      `bson:"envs" json:"envs"`
	ValueType    string             `bson:"value_type" json:"value_type"`
	Field        string             `bson:"field" json:"field"`
	DefaultValue interface{}        `bson:"default_value" json:"default_value"`
	ErrorValue   interface{}        `bson:"error_value" json:"error_value"`
	Enabled      bool               `bson:"enabled" json:"enabled"`
}
