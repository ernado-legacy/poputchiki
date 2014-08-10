package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (db *DB) AddLike(coll *mgo.Collection, user bson.ObjectId, target bson.ObjectId) error {
	if err := coll.UpdateId(target, bson.M{"$addToSet": bson.M{"liked_users": user}}); err != nil {
		return err
	}

	var likersID []bson.ObjectId
	if err := coll.FindId(target).Distinct("liked_users", &likersID); err != nil {
		return err
	}
	return coll.UpdateId(target, bson.M{"$set": bson.M{"likes": len(likersID)}})
}

func (db *DB) RemoveLike(coll *mgo.Collection, user bson.ObjectId, target bson.ObjectId) error {
	if err := coll.UpdateId(target, bson.M{"$pull": bson.M{"liked_users": user}}); err != nil {
		return err
	}

	var likersID []bson.ObjectId
	if err := coll.FindId(target).Distinct("liked_users", &likersID); err != nil {
		return err
	}
	return coll.UpdateId(target, bson.M{"$set": bson.M{"likes": len(likersID)}})
}

func (db *DB) GetLikes(coll *mgo.Collection, id bson.ObjectId) []*models.User {
	var likersID []bson.ObjectId
	var likers []*models.User

	err := coll.FindId(id).Distinct("liked_users", &likersID)
	if err != nil {
		return nil
	}

	err = db.users.Find(bson.M{"_id": bson.M{"$in": likersID}}).All(&likers)

	if err != nil {
		return nil
	}

	return likers
}
