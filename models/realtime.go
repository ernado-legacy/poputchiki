package models

import (
	"labix.org/v2/mgo/bson"
	"time"
)

type Message struct {
	Id          bson.ObjectId `json:"id"          bson:"_id"`
	Chat        bson.ObjectId `bson:"chat"`
	User        bson.ObjectId `json:"user"        bson:"user"`
	Origin      bson.ObjectId `json:"origin"      bson:"origin"`
	Destination bson.ObjectId `json:"destination" bson:"destination"`
	Time        time.Time     `json:"time"        bson:"time"`
	Text        string        `json:"text"        bson:"text"`
}

type RealtimeEvent struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
	Time time.Time   `json:"time"`
}

type ProgressMessage struct {
	Id       bson.ObjectId `json:"id,omitempty"`
	Progress float32       `json:"progress"`
}

type MessageSendBlacklisted struct {
	Id bson.ObjectId `json:"id"`
}
