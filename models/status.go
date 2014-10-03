package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Status struct {
	Id         bson.ObjectId   `json:"id,omitempty"     bson:"_id"`
	User       bson.ObjectId   `json:"user,omitempty"   bson:"user"`
	UserObject *User           `json:"user_object,omitempty" bson:"-"`
	Time       time.Time       `json:"time,omitempty"   bson:"time"`
	Text       string          `json:"text"             bson:"text"`
	ImageUrl   string          `json:"url,omitempty"    bson:"-"`
	Likes      int             `json:"likes,omitempty"  bson:"likes"`
	LikedUsers []bson.ObjectId `json:"liked_users"      bson:"liked_users"`
}

func (u *Status) Prepare(context Context) (err error) {
	if len(u.LikedUsers) == 0 {
		u.LikedUsers = []bson.ObjectId{}
	}
	if u.UserObject == nil {
		u.UserObject = context.DB.Get(u.User)
	}

	u.UserObject.Prepare(context)
	u.UserObject.CleanPrivate()
	u.ImageUrl = u.UserObject.AvatarUrl

	return err
}
