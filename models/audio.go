package models

import (
	"github.com/ernado/weed"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

type Audio struct {
	Id          bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User        bson.ObjectId `json:"user"                  bson:"user"`
	AudioAac    string        `json:"-"                     bson:"audio_aac"`
	AudioOgg    string        `json:"-"                     bson:"audio_ogg"`
	AudioUrl    string        `json:"url"                   bson:"-"`
	Description string        `json:"description,omitempty" bson:"description,omitempty"`
	Time        time.Time     `json:"time"                  bson:"time"`
	Duration    int64         `json:"duration"              bson:"duration"`
}

func (audio *Audio) Prepare(adapter *weed.Adapter, _ WebpAccept, _ VideoAccept, a AudioAccept) error {
	var err error
	log.Println(a, audio.AudioAac, audio.AudioOgg)
	audio.AudioUrl, err = adapter.GetUrl(audio.AudioAac)
	if a == AaOgg {
		log.Println("OGG")
		audio.AudioUrl, err = adapter.GetUrl(audio.AudioOgg)
	} else {
		log.Println("AAC")
	}

	return err
}

func (audio Audio) Url() string {
	return audio.AudioUrl
}
