package config

import "go.mongodb.org/mongo-driver/bson/primitive"

type GlobalSettings struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ScoreColors  []ScoreColor       `bson:"score_colors" json:"score_colors"`
	ScoreWeights []ScoreWeight      `bson:"score_weights" json:"score_weights"`
}

type ScoreColor struct {
	Label string `bson:"label" json:"label"`
	Range []int  `bson:"range" json:"range"`
	Color string `bson:"color" json:"color"`
}

type ScoreWeight struct {
	Weight     int    `bson:"weight" json:"weight"`
	Field      string `bson:"field" json:"field"`
	Comparator string `bson:"comparator" json:"comparator"`
	Arg        string `bson:"arg" json:"arg"`
}
