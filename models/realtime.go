package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Message struct {
	Id          bson.ObjectId `json:"id"          bson:"_id"`
	Chat        bson.ObjectId `json:"chat"        bson:"chat"`
	User        bson.ObjectId `json:"-"           bson:"user"`
	Origin      bson.ObjectId `json:"origin"      bson:"origin"`
	Destination bson.ObjectId `json:"destination" bson:"destination"`
	Read        bool          `json:"read"        bson:"read"`
	Time        time.Time     `json:"time"        bson:"time"`
	Text        string        `json:"text"        bson:"text"`
	Invite      bool          `json:"invite"      bson:"invite"`
}

func NewMessagePair(origin, destination bson.ObjectId, text string) (toOrigin, toDestination *Message) {
	toOrigin = new(Message)
	toDestination = new(Message)
	toOrigin.Id = bson.NewObjectId()
	toDestination.Id = bson.NewObjectId()
	toOrigin.Time = time.Now()
	toDestination.Time = toOrigin.Time
	toOrigin.User = origin
	toDestination.User = destination
	toOrigin.Chat = destination
	toDestination.Chat = origin
	toOrigin.Origin = origin
	toDestination.Origin = origin
	toOrigin.Destination = destination
	toDestination.Destination = destination
	toOrigin.Text = text
	toDestination.Text = text
	toOrigin.Read = true
	return
}

type RealtimeEvent struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
	Time time.Time   `json:"time"`
}

type Dialog struct {
	Id         bson.ObjectId `json:"id"     bson:"_id,omitempty"`
	Time       time.Time     `json:"time"   bson:"time"`
	Text       string        `json:"text"   bson:"text"`
	Origin     bson.ObjectId `json:"-"      bson:"origin,omitempty"`
	User       *User         `json:"user"`
	OriginUser *User         `json:"origin"`
}

type UnreadCount struct {
	Count int `json:"count"`
}

type ProgressMessage struct {
	Id       bson.ObjectId `json:"id,omitempty"`
	Progress float32       `json:"progress"`
}

type MessageSendBlacklisted struct {
	Id bson.ObjectId `json:"id"`
}
