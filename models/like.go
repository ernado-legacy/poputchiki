package models

import (
	"labix.org/v2/mgo/bson"
	"time"
)

type Like struct {
	Id     string        `json:"id"     bson:"_id"`
	User   bson.ObjectId `json:"user"   bson:"user"`
	Target bson.ObjectId `json:"target" bson:"target"`
	Time   time.Time     `json:"time"   bson:"time"`
}
