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

func (db *DB) UpdateAudioAAC(id bson.ObjectId, fid string) error {
	return db.audio.UpdateId(id, bson.M{"$set": bson.M{"audio_aac": fid}})
}

func (db *DB) UpdateAudioOGG(id bson.ObjectId, fid string) error {
	return db.audio.UpdateId(id, bson.M{"$set": bson.M{"audio_ogg": fid}})
}
