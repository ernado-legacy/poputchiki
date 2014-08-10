package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type File struct {
	Id   bson.ObjectId `json:"id,omitempty"  bson:"_id,omitempty"`
	Fid  string        `json:"fid"           bson:"fid"`
	User bson.ObjectId `json:"user"          bson:"user"`
	Time time.Time     `json:"time"          bson:"time"`
	Type string        `json:"type"          bson:"type"`
	Size int64         `json:"size"          bson:"size"`
	Url  string        `json:"url,omitempty" bson:"-"`
}
