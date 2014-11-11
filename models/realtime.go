package models

import (
	"log"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Message struct {
	Id          bson.ObjectId `json:"id"                     bson:"_id"`
	Chat        bson.ObjectId `json:"chat"                   bson:"chat"`
	User        bson.ObjectId `json:"-"                      bson:"user"`
	Origin      bson.ObjectId `json:"origin"                 bson:"origin"`
	Destination bson.ObjectId `json:"destination"            bson:"destination"`
	Read        bool          `json:"read"                   bson:"read"`
	Time        time.Time     `json:"time"                   bson:"time"`
	Text        string        `json:"text"                   bson:"text"`
	Invite      bool          `json:"invite"                 bson:"invite"`
	Photo       string        `json:"photo"                  bson:"photo"`
	PhotoUrl    string        `json:"photo_url"              bson:"photo_url"`
	LastMessage bson.ObjectId `json:"last_message,omitempty" bson:"-"`
}

type Messages []*Message

func (messages Messages) Prepare(context Context) error {
	for _, message := range messages {
		if err := message.Prepare(context); err != nil {
			return err
		}
	}
	return nil
}

// Prepare sets the url for attachment photo
func (m *Message) Prepare(context Context) error {
	if len(m.Photo) == 0 {
		return nil
	}
	url, err := context.Storage.URL(m.Photo)
	if err != nil {
		return err
	}
	m.PhotoUrl = url
	return nil
}

type Broadcast struct {
	Text string `json:"text" bson:"text"`
}

type Invite Message

func NewInvites(db DataBase, origin, destination bson.ObjectId, textOrigin, textDestination string) (toOrigin, toDestination *Invite) {
	m1, m2 := newMessagePair(db, origin, destination, "", "", true)
	m1.Text = textOrigin
	m2.Text = textDestination
	atoOrigin := Invite(*m1)
	atoDestination := Invite(*m2)
	return &atoOrigin, &atoDestination
}

func NewMessagePair(db DataBase, origin, destination bson.ObjectId, photo, text string) (toOrigin, toDestination *Message) {
	return newMessagePair(db, origin, destination, photo, text, false)
}

func newMessagePair(db DataBase, origin, destination bson.ObjectId, photo, text string, invite bool) (toOrigin, toDestination *Message) {
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
	toOrigin.Read = false
	toOrigin.Photo = photo
	toDestination.Photo = photo

	lastOrigin, err := db.GetLastMessageIdFromUser(origin, destination)
	if err != nil {
		log.Println(err)
	}
	lastDestination, err := db.GetLastMessageIdFromUser(destination, origin)
	if err != nil {
		log.Println(err)
	}
	toOrigin.LastMessage = lastOrigin
	toDestination.LastMessage = lastDestination

	toOrigin.Invite = invite
	toDestination.Invite = invite

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
	Unread     int           `json:"unread" bson:"unread"`
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
