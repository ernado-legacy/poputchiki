package models

import (
	"fmt"
	"github.com/ernado/weed"
	"gopkg.in/mgo.v2/bson"
	"log"
	"reflect"
	"strings"
	"time"
)

const (
	UpdateLikes  = "likes"
	UpdateGuests = SubscriptionGuests
)

type Update struct {
	Id          bson.ObjectId `json:"id"                    bson:"_id"`
	Destination bson.ObjectId `json:"-"                     bson:"destination"`
	User        bson.ObjectId `json:"user"                  bson:"user"`
	UserObject  *User         `json:"user_object,omitempty" bson:"-"`
	ImageWebp   string        `json:"-"                     bson:"image_webp"`
	ImageJpeg   string        `json:"-"                     bson:"image_jpeg"`
	ImageUrl    string        `json:"image_url,omitempty"   bson:"-"`
	Read        bool          `json:"read"                  bson:"read"`
	Type        string        `json:"type,omitempty"        bson:"type,omitempty"`
	TargetType  string        `json:"target_type,omitempty" bson:"target_type,omitempty"`
	Url         string        `json:"url,omitempty"         bson:"-"`
	Target      interface{}   `json:"target,omitempty"      bson:"target,omitempty"`
	Time        time.Time     `json:"time"                  bson:"time"`
}

type UpdateCounter struct {
	Type  string `json:"type"  bson:"_id"`
	Count int    `json:"count" bson:"count"`
}

func NewUpdate(destination, user bson.ObjectId, updateType string, media interface{}) Update {
	u := new(Update)
	u.Id = bson.NewObjectId()
	u.Destination = destination
	u.User = user
	u.Type = updateType
	u.Time = time.Now()
	if u.Type == UpdateLikes && media != nil {
		u.Target = media
		u.TargetType = strings.ToLower(reflect.TypeOf(media).Elem().Name())
	}
	return *u
}

func GetEventType(updateType string, media interface{}) string {
	if media == nil {
		return updateType
	}
	if updateType == SubscriptionInvites || updateType == SubscriptionMessages {
		return updateType
	}
	return fmt.Sprintf("%s_%s", updateType, strings.ToLower(reflect.TypeOf(media).Elem().Name()))
}

func (stripe *Update) Prepare(db DataBase, adapter *weed.Adapter, webp WebpAccept, video VideoAccept, audio AudioAccept) error {
	var media PrepareInterface
	var hasMedia bool = false

	if webp {
		stripe.ImageUrl, _ = adapter.GetUrl(stripe.ImageWebp)
	} else {
		stripe.ImageUrl, _ = adapter.GetUrl(stripe.ImageJpeg)
	}

	stripe.UserObject = db.Get(stripe.User)
	stripe.UserObject.Prepare(adapter, db, webp, audio)
	stripe.UserObject.CleanPrivate()

	switch stripe.Type {
	case "video":
		v := new(Video)
		convert(stripe.Target, v)
		media = v
	case "photo":
		v := new(Photo)
		convert(stripe.Target, v)
		media = v
	default:
		hasMedia = false
	}
	if hasMedia {
		if err := media.Prepare(adapter, webp, video, audio); err != nil {
			log.Println(err)
		}
		stripe.Url = media.Url()
	}
	return nil
}
