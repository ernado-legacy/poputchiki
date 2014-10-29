package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type PresentEvent struct {
	Id          bson.ObjectId `json:"id" bson:"_id"`
	Type        string        `json:"type" bson:"type"`
	Url         string        `json:"url"`
	Time        time.Time     `json:"time" bson:"time"`
	Origin      bson.ObjectId `json:"origin" bson:"origin"`
	Destination bson.ObjectId `json:"destination" bson:"destination"`
	Text        string        `json:"text" bson:"text"`
}

type Present struct {
	Id       bson.ObjectId `json:"id" bson:"_id"`
	Title    string        `json:"title" bson:"title"`
	Image    string        `bson:"image"`
	Url      string        `json:"url"`
	Cost     int           `json:"cost" bson:"cost"`
	Category string        `json:"category" bson:"category"`
	Time     time.Time     `bson:"time"`
}

func (p *Present) Prepare(context Context) (err error) {
	p.Url, err = context.Storage.URL(p.Image)
	return
}

type Presents []*Present

func (presents Presents) Prepare(context Context) error {
	for _, p := range presents {
		if err := p.Prepare(context); err != nil {
			return err
		}
	}
	return nil
}
