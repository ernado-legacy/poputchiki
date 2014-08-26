package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
)

func (db *DB) GetAudio(id bson.ObjectId) *models.Audio {
	a := new(models.Audio)
	if db.audio.FindId(id).One(a) != nil {
		return nil
	}
	return a
}

func (db *DB) AddAudio(audio *models.Audio) (*models.Audio, error) {
	if err := db.audio.Insert(audio); err != nil {
		return nil, err
	}
	return audio, db.users.UpdateId(audio.User, bson.M{"$set": bson.M{"audio": audio.Id}})
}

func (db *DB) UpdateAudioAAC(id bson.ObjectId, fid string) error {
	update := bson.M{"$set": bson.M{"audio_aac": fid}}
	if err := db.audio.UpdateId(id, update); err != nil {
		return err
	}
	_, err := db.users.UpdateAll(bson.M{"audio": id}, update)
	return err
}

func (db *DB) UpdateAudioOGG(id bson.ObjectId, fid string) error {
	update := bson.M{"$set": bson.M{"audio_ogg": fid}}
	if err := db.audio.UpdateId(id, update); err != nil {
		return err
	}
	_, err := db.users.UpdateAll(bson.M{"audio": id}, update)
	return err
}

func (db *DB) RemoveAudio(id bson.ObjectId) error {
	_, err := db.users.UpdateAll(bson.M{"audio": id}, bson.M{"$set": bson.M{"audio": ""}})
	if err != nil {
		return err
	}
	return db.audio.RemoveId(id)
}
