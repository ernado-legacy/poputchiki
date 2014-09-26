package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

func (db *DB) AddUpdate(destination, user bson.ObjectId, updateType string, media interface{}) (*models.Update, error) {
	u := models.NewUpdate(destination, user, updateType, media)
	log.Println("[db]", "adding update", u)
	return &u, db.updates.Insert(&u)
}

func (db *DB) SetUpdateRead(destination, id bson.ObjectId) error {
	query := bson.M{"destination": destination, "_id": id}
	update := bson.M{"$set": bson.M{"read": true}}
	return db.updates.Update(query, update)
}

func (db *DB) AddUpdateDirect(u *models.Update) (*models.Update, error) {
	log.Println("[db]", "adding update", u)
	return u, db.updates.Insert(u)
}

func (db *DB) GetUpdates(destination bson.ObjectId, t string, pagination models.Pagination) ([]*models.Update, error) {
	s := []*models.Update{}
	if pagination.Count == 0 {
		pagination.Count = searchCount
	}
	return s, db.updates.Find(bson.M{"destination": destination, "type": t}).Sort("-time").Skip(pagination.Offset).Limit(pagination.Count).All(&s)
}

func (db *DB) SetUpdatesRead(destination bson.ObjectId, t string) error {
	selector := bson.M{"destination": destination}
	if t != "" {
		selector = bson.M{"destination": destination, "type": t}
	}
	update := bson.M{"$set": bson.M{"read": true}}
	_, err := db.updates.UpdateAll(selector, update)
	return err
}

func (db *DB) IsUpdateDublicate(origin, destination bson.ObjectId, t string, duration time.Duration) (bool, error) {
	fromTime := time.Now().Add(-duration)
	query := bson.M{"time": bson.M{"$gte": fromTime}, "type": t, "user": origin, "destination": destination}
	count, err := db.updates.Find(query).Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) GetUpdatesTypeCount(destination bson.ObjectId, t string) (int, error) {
	return db.updates.Find(bson.M{"destination": destination, "type": t, "read": false}).Count()
}

func (db *DB) GetUpdatesCount(destination bson.ObjectId) ([]*models.UpdateCounter, error) {
	var result []*models.UpdateCounter
	match := bson.M{"$match": bson.M{"destination": destination, "read": false}}
	group := bson.M{"$group": bson.M{"_id": bson.M{"type": "$type"}, "count": bson.M{"$sum": 1}}}
	project := bson.M{"$project": bson.M{"_id": "$_id.type", "count": "$count"}}
	pipeline := []bson.M{match, group, project}
	pipe := db.updates.Pipe(pipeline)
	iter := pipe.Iter()
	if err := iter.All(&result); err != nil {
		return nil, err
	}
	all := &models.UpdateCounter{Type: "all"}
	for _, v := range result {
		all.Count += v.Count
	}
	result = append(result, all)
	return result, nil
}
