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
	if err := db.audio.Insert(audio); err != nil {
		return nil, err
	}
	return audio, db.users.UpdateId(audio.User, bson.M{"audio": audio.Id})
}

func (db *DB) UpdateAudioAAC(id bson.ObjectId, fid string) error {
	update := bson.M{"$set": bson.M{"audio_aac": fid}}
	if err := db.audio.UpdateId(id, update); err != nil {
		return err
	}
	return db.users.Update(bson.M{"audio": id}, update)
}

func (db *DB) UpdateAudioOGG(id bson.ObjectId, fid string) error {
	update := bson.M{"$set": bson.M{"audio_ogg": fid}}
	if err := db.audio.UpdateId(id, update); err != nil {
		return err
	}
	return db.users.Update(bson.M{"audio": id}, update)
}
