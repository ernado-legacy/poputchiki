package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Transaction struct {
	Id          int           `bson:"_id"`
	User        bson.ObjectId `bson:"user"`
	Value       int           `bson:"value"`
	Description string        `bson:"description"`
	Time        time.Time     `bson:"time"`
	Closed      bool          `bson:"closed"`
}
