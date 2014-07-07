package models

import (
	"labix.org/v2/mgo/bson"
	"time"
)

// additional user info
type UserInfo struct {
	Id    bson.ObjectId `json:"id"     bson:"_id"`
	About string        `json:"about"  bson:"about"`
}

func diff(t1, t2 time.Time) (years int) {
	t2 = t2.AddDate(0, 0, 1) // advance t2 to make the range inclusive

	for t1.AddDate(years, 0, 0).Before(t2) {
		years++
	}
	years--
	return
}

type Guest struct {
	Id    bson.ObjectId `json:"id"    bson:"_id"`
	User  bson.ObjectId `json:"user"  bson:"user"`
	Guest bson.ObjectId `json:"guest" bson:"guest"`
	Time  time.Time     `json:"time"  bson:"time"`
}

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

type LoginCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type File struct {
	Id   bson.ObjectId `json:"id,omitempty"  bson:"_id,omitempty"`
	Fid  string        `json:"fid"           bson:"fid"`
	User bson.ObjectId `json:"user"          bson:"user"`
	Time time.Time     `json:"time"          bson:"time"`
	Type string        `json:"type"          bson:"type"`
	Size int64         `json:"size"          bson:"size"`
	Url  string        `json:"url,omitempty" bson:"-"`
}

type StripeItemRequest struct {
	Id   bson.ObjectId `json:"id"`
	Type string        `json:"type"`
}

type Like struct {
	Id     string        `json:"id"     bson:"_id"`
	User   bson.ObjectId `json:"user"   bson:"user"`
	Target bson.ObjectId `json:"target" bson:"target"`
	Time   time.Time     `json:"time"   bson:"time"`
}

type Transaction struct {
	Id          int           `bson:"_id"`
	User        bson.ObjectId `bson:"user"`
	Value       int           `bson:"value"`
	Description string        `bson:"description"`
	Time        time.Time     `bson:"time"`
	Closed      bool          `bson:"closed"`
}
