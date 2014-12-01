package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Image struct {
	Id  bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Fid string        `json:"fid"          bson:"fid"`
	Url string        `json:"url"          bson:"url,omitempty"`
}

type Photo struct {
	Id            bson.ObjectId   `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId   `json:"user"                  bson:"user"`
	UserObject    *User           `json:"user_object,omitempty" bson:"-"`
	ImageWebp     string          `json:"-"                     bson:"image_webp"`
	ImageJpeg     string          `json:"-"                     bson:"image_jpeg"`
	ImageUrl      string          `json:"url"                   bson:"-"`
	ThumbnailWebp string          `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string          `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string          `json:"thumbnail_url"         bson:"-"`
	Description   string          `json:"description,omitempty" bson:"description,omitempty"`
	Hidden        bool            `json:"hidden,omitempty"      bson:"hidden,omitempty"`
	Likes         int             `json:"likes"                 bson:"likes"`
	LikedUsers    []bson.ObjectId `json:"liked_users"           bson:"liked_users"`
	Time          time.Time       `json:"time"                  bson:"time"`
}

func (p *Photo) Prepare(context Context) error {
	var err error
	p.ThumbnailUrl, err = context.Storage.URL(p.ThumbnailJpeg)
	if err != nil {
		return err
	}
	p.ImageUrl, err = context.Storage.URL(p.ImageJpeg)
	if len(p.LikedUsers) == 0 {
		p.LikedUsers = []bson.ObjectId{}
	}
	p.Likes = len(p.LikedUsers)
	return err
}

func (p Photo) Url() string {
	return p.ImageUrl
}
