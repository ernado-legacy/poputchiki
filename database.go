package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

type DB struct {
	users    *mgo.Collection
	guests   *mgo.Collection
	messages *mgo.Collection
}

func (db *DB) GetFavorites(id bson.ObjectId) []*User {
	var favoritesIds []bson.ObjectId
	var favorites []*User

	// first query - distinct favorites id's from user
	err := db.users.FindId(id).Distinct("favorites", &favoritesIds)

	if err != nil {
		return nil
	}

	// second query - get all users with favorites id's
	// query: db.users.find({_id: {$in: favoriteIds}})
	err = db.users.Find(bson.M{"_id": bson.M{"$in": favoritesIds}}).All(&favorites)

	if err != nil {
		return nil
	}

	return favorites
}

func (db *DB) Add(user *User) error {
	return db.users.Insert(user)
}

func (db *DB) Update(user *User) error {
	_, err := db.users.UpsertId(user.Id, &user)
	return err
}

func (db *DB) Get(id bson.ObjectId) *User {
	var u User
	err := db.users.FindId(id).One(&u)

	if err != nil {
		log.Println("getting user by id error:", err, id)
		return nil
	}

	return &u
}

// Get user by username (email)
func (db *DB) GetUsername(username string) *User {
	var u User
	err := db.users.Find(bson.M{"email": username}).One(&u)

	if err != nil {
		log.Println("getting user by username error:", err)
		return nil
	}
	return &u

}

func (db *DB) AddToFavorites(id bson.ObjectId, favId bson.ObjectId) error {
	var u User
	change := mgo.Change{Update: bson.M{"$addToSet": bson.M{"favorites": favId}}}

	_, err := db.users.FindId(id).Apply(change, &u)

	return err
}

func (db *DB) RemoveFromFavorites(id bson.ObjectId, favId bson.ObjectId) error {
	var u User
	change := mgo.Change{Update: bson.M{"$pull": bson.M{"favorites": favId}}}

	_, err := db.users.FindId(id).Apply(change, &u)

	return err
}

func (db *DB) AddGuest(id bson.ObjectId, guest bson.ObjectId) error {
	g := Guest{bson.NewObjectId(), id, guest, time.Now()}
	_, err := db.guests.Upsert(bson.M{"user": id, "guest": guest}, &g)
	return err
}

func (db *DB) GetAllGuests(id bson.ObjectId) ([]*User, error) {
	g := []bson.ObjectId{}
	u := []*User{}

	// first query - get all guests ids from guest-pair collection
	err := db.guests.Find(bson.M{"user": id}).Distinct("guest", &g)
	if err != nil {
		return nil, err
	}

	// second query - get all users with that id's
	err = db.users.Find(bson.M{"_id": bson.M{"$in": &g}}).All(&u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (db *DB) SendMessage(origin bson.ObjectId, destination bson.ObjectId, text string) error {
	t := time.Now()
	m1 := Message{bson.NewObjectId(), origin, origin, destination, t, text}
	m2 := Message{bson.NewObjectId(), destination, origin, destination, t, text}

	err := db.messages.Insert(&m1)
	if err != nil {
		return err
	}

	return db.messages.Insert(&m2)
}

func (db *DB) RemoveMessage(id bson.ObjectId) error {
	return db.messages.RemoveId(id)
}

func (db *DB) GetMessage(id bson.ObjectId) (message *Message, err error) {
	err = db.messages.FindId(id).One(message)
	return message, err
}

func (db *DB) GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) (messages []*Message, err error) {
	err = db.messages.Find(bson.M{"user": userReciever, "origin": userOrigin}).All(&messages)
	return messages, err
}
