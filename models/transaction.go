package models

import (
	"labix.org/v2/mgo/bson"
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
