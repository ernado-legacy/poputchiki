package models

import (
	"gopkg.in/mgo.v2/bson"
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

func (audio *Audio) Prepare(context Context) error {
	var err error
	audio.AudioUrl, err = context.Storage.URL(audio.AudioAac)
	if context.Audio == AaOgg {
		audio.AudioUrl, err = context.Storage.URL(audio.AudioOgg)
	}
	return err
}

func (audio Audio) Url() string {
	return audio.AudioUrl
}
