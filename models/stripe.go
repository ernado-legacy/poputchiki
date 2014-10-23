package models

import (
	"errors"
	"gopkg.in/mgo.v2/bson"
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
	Media      interface{}   `json:"-"                     bson:"media"`
	Countries  []string      `json:"countries,omitempty"   bson:"countries,omitempty"`
	Time       time.Time     `json:"time"                  bson:"time"`
	Text       string        `json:"text,omitempty" bson:"text,omitempty"`
}

type Stripe []*StripeItem

func (s Stripe) Prepare(context Context) error {
	for _, elem := range s {
		if err := elem.Prepare(context); err != nil {
			return err
		}
	}
	return nil
}

func convert(input interface{}, output interface{}) error {
	data, _ := bson.Marshal(input)
	return bson.Unmarshal(data, output)
}

func (stripe *StripeItem) Prepare(context Context) error {
	var err error
	if len(stripe.ImageJpeg) > 0 {
		stripe.ImageUrl, err = context.Storage.URL(stripe.ImageJpeg)
		if context.WebP {
			stripe.ImageUrl, err = context.Storage.URL(stripe.ImageWebp)
		}
		if err != nil {
			log.Println(err)
		}
	}
	stripe.UserObject = context.DB.Get(stripe.User)
	stripe.UserObject.Prepare(context)
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

	if err := media.Prepare(context); err != nil {
		log.Println(err)
	}
	stripe.Url = media.Url()
	return nil
}

type StripeItemRequest struct {
	Id    bson.ObjectId `json:"id"`
	Photo bson.ObjectId `json:"photo,omitempty"`
	Type  string        `json:"type"`
	Text  string        `json:"text"`
}
