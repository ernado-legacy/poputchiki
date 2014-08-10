package models

import (
	"encoding/json"
	"errors"
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

type StripeItem struct {
	Id        bson.ObjectId `json:"id"                  bson:"_id"`
	User      bson.ObjectId `json:"user"                bson:"user"`
	Name      string        `json:"name"                bson:"name"`
	Age       int           `json:"age,omitempty"       bson:"-"`
	ImageWebp string        `json:"-"                   bson:"image_webp"`
	ImageJpeg string        `json:"-"                   bson:"image_jpeg"`
	ImageUrl  string        `json:"image_url,omitempty" bson:"-"`
	Type      string        `json:"type"                bson:"type"`
	Url       string        `json:"url"                 bson:"-"`
	Media     interface{}   `json:"-"                   bson:"media"`
	Countries []string      `json:"countries,omitempty" bson:"countries,omitempty"`
	Time      time.Time     `json:"time"                bson:"time"`
}

func convert(input interface{}, output interface{}) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}

func (stripe *StripeItem) Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, audio AudioAccept) error {
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
	case "photo":
		v := new(Photo)
		convert(stripe.Media, v)
		media = v
	default:
		return errors.New("bad type")
	}

	return media.Prepare(adapter, webp, video, audio)
}

type StripeItemRequest struct {
	Id   bson.ObjectId `json:"id"`
	Type string        `json:"type"`
}
