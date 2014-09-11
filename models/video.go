package models

import (
	"github.com/ernado/weed"
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

func (v *Video) Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, _ AudioAccept) error {
	var err error
	v.VideoUrl, err = adapter.GetUrl(v.VideoMpeg)
	if video == VaWebm {
		v.VideoUrl, err = adapter.GetUrl(v.VideoWebm)
	}
	if err != nil {
		return err
	}
	if webp {
		v.ThumbnailUrl, err = adapter.GetUrl(v.ThumbnailWebp)
	} else {
		v.ThumbnailUrl, err = adapter.GetUrl(v.ThumbnailJpeg)
	}
	if len(v.LikedUsers) == 0 {
		v.LikedUsers = []bson.ObjectId{}
	}
	return err
}

func (v Video) Url() string {
	return v.VideoUrl
}
