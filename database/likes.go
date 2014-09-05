package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
)

func (db *DB) AddLike(coll *mgo.Collection, user bson.ObjectId, target bson.ObjectId) error {
	update := bson.M{"$addToSet": bson.M{"liked_users": user}}
	selector := bson.M{"_id": target}
	if err := coll.Update(selector, update); err != nil {
		return err
	}

	var likers []bson.ObjectId
	if err := coll.FindId(target).Distinct("liked_users", &likers); err != nil {
		return err
	}

	return coll.UpdateId(target, bson.M{"$set": bson.M{"likes": len(likers)}})
}

func (db *DB) RemoveLike(coll *mgo.Collection, user bson.ObjectId, target bson.ObjectId) error {
	if err := coll.UpdateId(target, bson.M{"$pull": bson.M{"liked_users": user}}); err != nil {
		return err
	}

	var likers []bson.ObjectId
	if err := coll.FindId(target).Distinct("liked_users", &likers); err != nil {
		return err
	}
	return coll.UpdateId(target, bson.M{"$set": bson.M{"likes": len(likers)}})
}

func (db *DB) GetLikes(coll *mgo.Collection, id bson.ObjectId) []*models.User {
	var likersID []bson.ObjectId
	var likers []*models.User
	err := coll.FindId(id).Distinct("liked_users", &likersID)
	if err != nil {
		log.Println(err)
		return nil
	}

	err = db.users.Find(bson.M{"_id": bson.M{"$in": likersID}}).All(&likers)

	if err != nil {
		log.Println(err)
		return nil
	}

	return likers
}
