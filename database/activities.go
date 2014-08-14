package database

import (
	. "github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

func (db *DB) AddActivity(user bson.ObjectId, key string) error {
	id := bson.NewObjectId()
	activity := UserActivity{id, user, key, time.Now()}
	return db.activities.Insert(activity)
}

func (db *DB) GetActivityCount(user bson.ObjectId, key string, duration time.Duration) (count int, err error) {
	query := bson.M{"user": user, "key": key, "time": bson.M{"$gte": time.Now().Add(-duration)}}
	log.Printf("%+v", query)
	return db.activities.Find(query).Count()
}
