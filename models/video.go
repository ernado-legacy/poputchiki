package models

import (
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"time"
)

type Video struct {
	Id            bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId `json:"user"                  bson:"user"`
	VideoWebm     string        `json:"-"                     bson:"video_webm"`
	VideoMpeg     string        `json:"-"                     bson:"video_mpeg"`
	VideoUrl      string        `json:"url"                   bson:"-"`
	ThumbnailWebp string        `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string        `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string        `json:"thumbnail_url"         bson:"-"`
	Description   string        `json:"description,omitempty" bson:"description,omitempty"`
	Time          time.Time     `json:"time"                  bson:"time"`
	Likes         int           `json:"likes"                 bson:"likes"`
	Duration      int64         `json:"duration"              bson:"duration"`
}

func (v Video) Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, _ AudioAccept) error {
	var err error
	if video == VA_WEBM {
		v.VideoUrl, err = adapter.GetUrl(v.VideoWebm)
	} else if video == VA_MP4 {
		v.VideoUrl, err = adapter.GetUrl(v.VideoMpeg)
	}
	if err != nil {
		return err
	}
	if webp {
		v.ThumbnailUrl, err = adapter.GetUrl(v.ThumbnailWebp)
	} else {
		v.ThumbnailUrl, err = adapter.GetUrl(v.ThumbnailJpeg)
	}
	return err
}
