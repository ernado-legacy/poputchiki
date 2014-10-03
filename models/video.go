package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Video struct {
	Id            bson.ObjectId   `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId   `json:"user"                  bson:"user"`
	VideoWebm     string          `json:"-"                     bson:"video_webm"`
	VideoMpeg     string          `json:"-"                     bson:"video_mpeg"`
	VideoUrl      string          `json:"url"                   bson:"-"`
	ThumbnailWebp string          `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string          `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string          `json:"thumbnail_url"         bson:"-"`
	Description   string          `json:"description,omitempty" bson:"description,omitempty"`
	Time          time.Time       `json:"time"                  bson:"time"`
	Likes         int             `json:"likes"                 bson:"likes"`
	LikedUsers    []bson.ObjectId `json:"liked_users"           bson:"liked_users,omitempty"`
	Duration      int64           `json:"duration"              bson:"duration"`
}

func (v *Video) Prepare(context Context) error {
	var err error
	v.VideoUrl, err = context.Storage.URL(v.VideoMpeg)
	if context.Video == VaWebm {
		v.VideoUrl, err = context.Storage.URL(v.VideoWebm)
	}
	if err != nil {
		return err
	}
	if context.WebP {
		v.ThumbnailUrl, err = context.Storage.URL(v.ThumbnailWebp)
	} else {
		v.ThumbnailUrl, err = context.Storage.URL(v.ThumbnailJpeg)
	}
	if len(v.LikedUsers) == 0 {
		v.LikedUsers = []bson.ObjectId{}
	}
	return err
}

func (v Video) Url() string {
	return v.VideoUrl
}
