package database

import (
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func (db *DB) GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) (messages []*models.Message, err error) {
	err = db.messages.Find(bson.M{"user": userReciever, "chat": userOrigin}).All(&messages)
	return messages, err
}

func (db *DB) AddMessage(m *models.Message) error {
	return db.messages.Insert(m)
}

func (db *DB) RemoveMessage(id bson.ObjectId) error {
	return db.messages.RemoveId(id)
}

func (db *DB) GetMessage(id bson.ObjectId) (*models.Message, error) {
	message := &models.Message{}
	err := db.messages.FindId(id).One(message)
	return message, err
}
func (db *DB) SetRead(query bson.M) error {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"read": true}}}
	_, err := db.messages.Find(query).Apply(change, nil)
	return err
}

func (db *DB) GetChats(id bson.ObjectId) ([]*models.User, error) {
	var users []*models.User
	var ids []bson.ObjectId
	var result []bson.M

	// preparing query
	match := bson.M{"$match": bson.M{"user": id}}
	sort := bson.M{"$sort": bson.M{"time": -1}}
	group := bson.M{"$group": bson.M{"_id": bson.M{"chat": "$chat"}}}
	project := bson.M{"$project": bson.M{"_id": "$_id.chat"}}
	pipeline := []bson.M{match, sort, group, project}
	pipe := db.messages.Pipe(pipeline)

	iter := pipe.Iter()
	iter.All(&result)

	// processing result
	for _, v := range result {
		switch uid := v["_id"].(type) {
		case bson.ObjectId:
			ids = append(ids, uid)
		default:
			continue
		}
	}

	return users, db.users.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&users)
}
