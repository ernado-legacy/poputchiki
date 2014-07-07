package models

import (
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"time"
)

type StatusUpdate struct {
	Id        bson.ObjectId `json:"id"            bson:"_id"`
	User      bson.ObjectId `json:"user"          bson:"user"`
	Name      string        `json:"name"          bson:"-"`
	Age       int           `json:"age,omitempty" bson:"-"`
	Time      time.Time     `json:"time"          bson:"time"`
	Text      string        `json:"text"          bson:"text"`
	ImageWebp string        `json:"-"             bson:"image_webp"`
	ImageJpeg string        `json:"-"             bson:"image_jpeg"`
	ImageUrl  string        `json:"url"           bson:"-"`
	Likes     int           `json:"likes"         bson:"likes"`
}

func (u StatusUpdate) Prepare(adapter *weed.Adapter, webp WebpAccept) (err error) {
	if webp {
		u.ImageUrl, err = adapter.GetUrl(u.ImageWebp)
	} else {
		u.ImageUrl, err = adapter.GetUrl(u.ImageJpeg)
	}
	return err
}
