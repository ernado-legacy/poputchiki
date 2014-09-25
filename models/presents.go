package models

import (
	"github.com/ernado/weed"
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

// func (e *PresentEvent) Prepare(adapter *weed.Adapter, webp WebpAccept, db DataBase) error {
// 	var err error
// 	if webp {
// 		p.ThumbnailUrl, err = adapter.GetUrl(p.ThumbnailWebp)
// 		if err != nil {
// 			return err
// 		}
// 		p.ImageUrl, err = adapter.GetUrl(p.ImageWebp)
// 	} else {
// 		p.ThumbnailUrl, err = adapter.GetUrl(p.ThumbnailJpeg)
// 		if err != nil {
// 			return err
// 		}
// 		p.ImageUrl, err = adapter.GetUrl(p.ImageJpeg)
// 	}
// 	if len(p.LikedUsers) == 0 {
// 		p.LikedUsers = []bson.ObjectId{}
// 	}
// 	return err
// }

type Present struct {
	Id    bson.ObjectId `json:"id" bson:"_id"`
	Title string        `json:"title" bson:"title"`
	Image string        `bson:"image"`
	Url   string        `json:"url"`
	Cost  int           `json:"cost" bson:"cost"`
	Time  time.Time     `bson:"time"`
}

func (p *Present) Prepare(adapter *weed.Adapter) (err error) {
	p.Url, err = adapter.GetUrl(p.Image)
	return
}

type Presents []*Present

func (presents Presents) Prepare(adapter *weed.Adapter) error {
	for _, p := range presents {
		if err := p.Prepare(adapter); err != nil {
			return err
		}
	}
	return nil
}
