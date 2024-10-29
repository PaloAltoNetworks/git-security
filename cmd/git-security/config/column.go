package config

import "go.mongodb.org/mongo-driver/bson/primitive"

type Column struct {
	ID                  primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	Type                string               `bson:"type" json:"type"`
	Title               string               `bson:"title" json:"title"`
	Description         string               `bson:"description" json:"description"`
	Key                 string               `bson:"key" json:"key"`
	Width               int                  `bson:"width" json:"width"`
	Show                bool                 `bson:"show" json:"show"`
	Filter              bool                 `bson:"filter" json:"filter"`
	FilterExpanded      bool                 `bson:"filter_expanded" json:"filter_expanded"`
	CSV                 bool                 `bson:"csv" json:"csv"`
	Order               string               `bson:"order" json:"order"`
	GroupBy             string               `bson:"group_by" json:"group_by"`
	GroupByColumnStyles []GroupByColumnStyle `bson:"group_by_column_styles" json:"group_by_column_styles"`
}

type GroupByColumnStyle struct {
	Color      string `bson:"color" json:"color"`
	Value      string `bson:"value" json:"value"`
	Comparator string `bson:"comparator" json:"comparator"`
	Arg        string `bson:"arg" json:"arg"`
}
