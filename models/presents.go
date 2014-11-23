package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type PresentEvent struct {
	Id          bson.ObjectId `json:"id" bson:"_id"`
	Type        string        `json:"type" bson:"type"`
	Url         string        `json:"url"`
	Time        time.Time     `json:"time" bson:"time"`
	Origin      bson.ObjectId `json:"origin" bson:"origin"`
	OriginUser  *User         `json:"origin_user"`
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

func (p *PresentEvent) PrepareContext(context Context, presents Presents) (err error) {
	p.Url = presents.Url(p.Type)
	p.OriginUser = context.DB.Get(p.Origin)
	if p.OriginUser != nil {
		return p.OriginUser.Prepare(context)
	}
	return nil
}

func (events PresentEvents) Prepare(context Context) (err error) {
	presents, err := context.DB.GetAllPresents()
	if err != nil {
		return err
	}
	if err = Presents(presents).Prepare(context); err != nil {
		return err
	}
	for _, p := range events {
		if err := p.PrepareContext(context, presents); err != nil {
			return err
		}
	}
	return nil
}

func (p *Present) Prepare(context Context) (err error) {
	p.Url, err = context.Storage.URL(p.Image)
	return
}

type Presents []*Present
type PresentEvents []*PresentEvent

func (presents Presents) Prepare(context Context) error {
	for _, p := range presents {
		if err := p.Prepare(context); err != nil {
			return err
		}
	}
	return nil
}

func (presents Presents) Url(title string) (url string) {
	for _, p := range presents {
		if p.Title == title {
			return p.Url
		}
	}
	return
}
