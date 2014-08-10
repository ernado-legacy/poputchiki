package models

import (
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
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
	if a == AaAac {
		audio.AudioUrl, err = adapter.GetUrl(audio.AudioAac)
	} else if a == AaOgg {
		audio.AudioUrl, err = adapter.GetUrl(audio.AudioOgg)
	}
	return err
}

func (audio Audio) Url() string {
	return audio.AudioUrl()
}
