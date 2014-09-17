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
	UpdateLikes    = "likes"
	UpdateGuests   = SubscriptionGuests
	UpdateMessages = SubscriptionMessages
)

type Update struct {
	Id          bson.ObjectId `json:"id"                    bson:"_id"`
	Destination bson.ObjectId `json:"destination,omitempty" bson:"destination"`
	User        bson.ObjectId `json:"user"                  bson:"user"`
	UserObject  *User         `json:"user_object,omitempty" bson:"-"`
	ImageWebp   string        `json:"image_webp,omitempty"  bson:"image_webp,omitempty"`
	ImageJpeg   string        `json:"image_jpeg,omitempty"  bson:"image_jpeg,omitempty"`
	ImageUrl    string        `json:"image_url,omitempty"   bson:"-"`
	Read        bool          `json:"read"                  bson:"read"`
	Type        string        `json:"type,omitempty"        bson:"type,omitempty"`
	TargetType  string        `json:"target_type,omitempty" bson:"target_type,omitempty"`
	Url         string        `json:"url,omitempty"         bson:"-"`
	Target      interface{}   `json:"target,omitempty"      bson:"target,omitempty"`
	Time        time.Time     `json:"time"                  bson:"time"`
}

func (u *Update) Theme() (theme string) {
	if u.Type == "messages" {
		theme = fmt.Sprintf("Пользователь %s прислал вам сообщение", u.UserObject.Name)
	}
	if u.Type == "guests" {
		theme = fmt.Sprintf("Пользователь %s заходил на вашу страницу", u.UserObject.Name)
	}

	if u.Type == "likes" {
		var t string
		if u.TargetType == "video" {
			t = "ваше видео"
		}
		if u.TargetType == "photo" {
			t = "ваше фото"
		}
		if u.TargetType == "status" {
			t = "ваш статус"
		}

		theme = fmt.Sprintf("Пользователь %s оценил %s", u.UserObject.Name, t)
	}

	return
}

type UpdateCounter struct {
	Type  string `json:"type"  bson:"_id"`
	Count int    `json:"count" bson:"count"`
}

type Counters []*UpdateCounter

func NewUpdate(destination, user bson.ObjectId, updateType string, media interface{}) Update {
	u := new(Update)
	u.Id = bson.NewObjectId()
	u.Destination = destination
	u.User = user
	u.Type = updateType
	u.Time = time.Now()
	if media != nil {
		u.TargetType = strings.ToLower(reflect.TypeOf(media).Elem().Name())
		u.Target = media
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
	log.Println("[prepare]", "preparing update")
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
	log.Println("[prepare]", "prepared")
	return nil
}
