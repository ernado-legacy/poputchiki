package models

import (
	"errors"
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

type StripeItem struct {
	Id         bson.ObjectId `json:"id"                    bson:"_id"`
	User       bson.ObjectId `json:"user"                  bson:"user"`
	UserObject *User         `json:"user_object,omitempty" bson:"-"`
	Name       string        `json:"name"                  bson:"name"`
	Age        int           `json:"age,omitempty"         bson:"-"`
	ImageWebp  string        `json:"-"                     bson:"image_webp"`
	ImageJpeg  string        `json:"-"                     bson:"image_jpeg"`
	ImageUrl   string        `json:"image_url,omitempty"   bson:"-"`
	Type       string        `json:"type"                  bson:"type"`
	Url        string        `json:"url,omitemptry"        bson:"-"`
	Media      interface{}   `json:"media"                 bson:"media"`
	Countries  []string      `json:"countries,omitempty"   bson:"countries,omitempty"`
	Time       time.Time     `json:"time"                  bson:"time"`
}

func convert(input interface{}, output interface{}) error {
	data, _ := bson.Marshal(input)
	return bson.Unmarshal(data, output)
}

func (stripe *StripeItem) Prepare(db DataBase, adapter *weed.Adapter, webp WebpAccept, video VideoAccept, audio AudioAccept) error {
	log.Printf("%+v", stripe)

	var err error
	if webp {
		stripe.ImageUrl, err = adapter.GetUrl(stripe.ImageWebp)
	} else {
		stripe.ImageUrl, err = adapter.GetUrl(stripe.ImageJpeg)
	}
	if err != nil {
		log.Println(err)
		// return err
	}
	stripe.UserObject = db.Get(stripe.User)
	stripe.UserObject.Prepare(adapter, db, webp, audio)
	stripe.Age = stripe.UserObject.Age
	stripe.Name = stripe.UserObject.Name
	stripe.UserObject.CleanPrivate()

	var media PrepareInterface
	switch stripe.Type {
	case "video":
		v := new(Video)
		convert(stripe.Media, v)
		media = v
	case "audio":
		v := new(Audio)
		convert(stripe.Media, v)
		media = v
		stripe.ImageUrl = stripe.UserObject.AvatarUrl
	case "photo":
		v := new(Photo)
		convert(stripe.Media, v)
		media = v
	default:
		return errors.New("bad type")
	}

	if err := media.Prepare(adapter, webp, video, audio); err != nil {
		log.Println(err)
		// return err
	}
	stripe.Url = media.Url()
	return nil
}

type StripeItemRequest struct {
	Id   bson.ObjectId `json:"id"`
	Type string        `json:"type"`
}
