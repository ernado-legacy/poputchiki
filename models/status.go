package models

import (
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"time"
)

type StatusUpdate struct {
	Id         bson.ObjectId   `json:"id,omitempty"     bson:"_id"`
	User       bson.ObjectId   `json:"user,omitempty"   bson:"user"`
	Name       string          `json:"name,omitempty"   bson:"-"`
	Birthday   time.Time       `json:"-"`
	Age        int             `json:"age,omitempty"    bson:"-"`
	Time       time.Time       `json:"time,omitempty"   bson:"time"`
	Text       string          `json:"text,omitempty"   bson:"text"`
	ImageWebp  string          `json:"-"                bson:"image_webp"`
	ImageJpeg  string          `json:"-"                bson:"image_jpeg"`
	ImageUrl   string          `json:"url,omitempty"    bson:"-"`
	Likes      int             `json:"likes,omitempty"  bson:"likes"`
	LikedUsers []bson.ObjectId `json:"liked_users"      bson:"liked_users"`
}

func (u StatusUpdate) Prepare(adapter *weed.Adapter, webp WebpAccept) (err error) {
	if webp {
		u.ImageUrl, err = adapter.GetUrl(u.ImageWebp)
	} else {
		u.ImageUrl, err = adapter.GetUrl(u.ImageJpeg)
	}
	if len(u.LikedUsers) == 0 {
		u.LikedUsers = []bson.ObjectId{}
	}

	now := time.Now()
	defer func() {
		if r := recover(); r != nil {
			u.Birthday = time.Unix(0, 0)
		}
	}()
	if u.Birthday.Unix() != 0 {
		u.Age = diff(u.Birthday, now)
	}

	return err
}
