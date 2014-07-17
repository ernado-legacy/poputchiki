package database

import (
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

func (db *DB) NewConfirmationToken(id bson.ObjectId) *models.EmailConfirmationToken {
	return db.NewConfirmationTokenValue(id, gotok.Generate(id).Token)
}

func (db *DB) GetConfirmationToken(token string) *models.EmailConfirmationToken {
	t := &models.EmailConfirmationToken{}
	selector := bson.M{"token": token}
	// log.Println("[GetConfirmationToken]", "searching", token)
	if err := db.conftokens.Find(selector).One(t); err != nil {
		// log.Println("[GetConfirmationToken]", err)
		return nil
	}

	if err := db.conftokens.Remove(selector); err != nil {
		log.Println("[GetConfirmationToken]", "[remove]", err)
		return nil
	}
	return t
}

func (db *DB) NewConfirmationTokenValue(id bson.ObjectId, token string) *models.EmailConfirmationToken {
	t := &models.EmailConfirmationToken{}
	t.Id = bson.NewObjectId()
	t.User = id
	t.Time = time.Now()
	t.Token = token
	err := db.conftokens.Insert(t)
	// log.Println("[NewConfirmationToken]", "inserted", t.Token)
	if err != nil {
		log.Println("[NewConfirmationToken]", err)
		return nil
	}
	return t
}
