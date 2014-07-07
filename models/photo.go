package models

import (
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"time"
)

type Image struct {
	Id  bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Fid string        `json:"fid"          bson:"fid"`
	Url string        `json:"url"          bson:"url,omitempty"`
}

type Photo struct {
	Id            bson.ObjectId   `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId   `json:"user"                  bson:"user"`
	ImageWebp     string          `json:"-"                     bson:"image_webp"`
	ImageJpeg     string          `json:"-"                     bson:"image_jpeg"`
	ImageUrl      string          `json:"url"                   bson:"-"`
	ThumbnailWebp string          `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string          `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string          `json:"thumbnail_url"         bson:"-"`
	Description   string          `json:"description,omitempty" bson:"description,omitempty"`
	Likes         int             `json:"likes"                 bson:"likes"`
	Time          time.Time       `json:"time"                  bson:"time"`
	LikedUsers    []bson.ObjectId `bson:"liked_users"`
}

func (p Photo) Prepare(adapter *weed.Adapter, webp WebpAccept, _ VideoAccept, _ AudioAccept) error {
	var err error
	if webp {
		p.ThumbnailUrl, err = adapter.GetUrl(p.ThumbnailWebp)
		if err != nil {
			return err
		}
		p.ImageUrl, err = adapter.GetUrl(p.ImageWebp)
	} else {
		p.ThumbnailUrl, err = adapter.GetUrl(p.ThumbnailJpeg)
		if err != nil {
			return err
		}
		p.ImageUrl, err = adapter.GetUrl(p.ImageJpeg)
	}
	return err
}
