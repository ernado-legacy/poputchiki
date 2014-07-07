package database

import (
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo/bson"
)

func (db *DB) GetAudio(id bson.ObjectId) *models.Audio {
	a := &models.Audio{}
	if db.audio.FindId(id).One(a) != nil {
		return nil
	}
	return a
}

func (db *DB) AddAudio(audio *models.Audio) (*models.Audio, error) {
	return audio, db.audio.Insert(audio)
}
