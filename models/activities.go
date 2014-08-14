package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Activity struct {
	Key    string
	Weight float64
	Count  int
}

type UserActivity struct {
	Id   bson.ObjectId `bson:"id"`
	User bson.ObjectId `bson:"user"`
	Key  string        `bson:"key"`
	Time time.Time     `bson:"time"`
}
