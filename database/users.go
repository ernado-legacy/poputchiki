package database

import (
	. "github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"sort"
	"time"
)

func (db *DB) Add(user *User) error {
	return db.users.Insert(user)
}

func (db *DB) Update(id bson.ObjectId, update bson.M) (*User, error) {
	u := &User{}
	change := mgo.Change{Update: bson.M{"$set": update}}
	_, err := db.users.FindId(id).Apply(change, u)
	return u, err
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

func (db *DB) AddGuest(id bson.ObjectId, guest bson.ObjectId) error {
	g := Guest{}
	g.Time = time.Now()
	g.Guest = guest
	g.User = id
	_, err := db.guests.Upsert(bson.M{"user": g.User, "guest": g.Guest}, &g)
	return err
}

func (db *DB) AddToBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error {
	var u User
	change := mgo.Change{Update: bson.M{"$addToSet": bson.M{"blacklist": blacklisted}}}

	_, err := db.users.FindId(id).Apply(change, &u)

	return err
}

func (db *DB) RemoveFromBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error {
	var u User
	change := mgo.Change{Update: bson.M{"$pull": bson.M{"blacklist": blacklisted}}}

	_, err := db.users.FindId(id).Apply(change, &u)

	return err
}

func (db *DB) GetAllGuests(id bson.ObjectId) ([]*User, error) {
	g := []Guest{}
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

type guestByTime []*GuestUser

func (a guestByTime) Len() int {
	return len(a)
}

func (a guestByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a guestByTime) Less(i, j int) bool {
	return a[i].Time.Before(a[j].Time)
}

func (db *DB) GetAllGuestUsers(id bson.ObjectId) ([]*GuestUser, error) {
	var ids []bson.ObjectId
	var result []Guest
	times := make(map[bson.ObjectId]time.Time)
	var guests []*GuestUser
	var buff []*User

	err := db.guests.Find(bson.M{"user": id}).Sort("-time").All(&result)
	if err != nil {
		return nil, err
	}

	for k := range result {
		ids = append(ids, result[k].Guest)
		times[result[k].Guest] = result[k].Time
	}

	err = db.users.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&buff)
	if err != nil {
		return nil, err
	}

	for k := range buff {
		user := *buff[k]
		guest := &GuestUser{User: user, Time: times[user.Id]}
		guests = append(guests, guest)
	}
	sort.Sort(guestByTime(guests))
	return guests, nil
}

func (db *DB) SetOnlineStatus(id bson.ObjectId, status bool) error {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"online": status}}}
	_, err := db.users.FindId(id).Apply(change, nil)
	return err
}

func (db *DB) SetOnline(id bson.ObjectId) error {
	return db.SetOnlineStatus(id, true)
}

func (db *DB) SetOffline(id bson.ObjectId) error {
	return db.SetOnlineStatus(id, false)
}

func (db *DB) ChangeBalance(id bson.ObjectId, delta int) error {
	change := mgo.Change{Update: bson.M{"$inc": bson.M{"balance": delta}}}

	// integer overflow / negative balance protection
	query := bson.M{"_id": id, "balance": bson.M{"$gte": (-1) * delta}}
	if delta > 0 {
		query = bson.M{"_id": id}
	}
	_, err := db.users.Find(query).Apply(change, &User{})
	return err
}

func (db *DB) IncBalance(id bson.ObjectId, amount uint) error {
	return db.ChangeBalance(id, int(amount))
}

func (db *DB) DecBalance(id bson.ObjectId, amount uint) error {
	return db.ChangeBalance(id, (-1)*int(amount))
}

func (db *DB) SetLastActionNow(id bson.ObjectId) error {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"lastaction": time.Now()}}}
	_, err := db.users.FindId(id).Apply(change, nil)
	return err
}

func (db *DB) SetAvatar(user, avatar bson.ObjectId) error {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"avatar": avatar}}}
	_, err := db.users.FindId(user).Apply(change, &bson.M{})
	return err
}

func (db *DB) Search(q *SearchQuery, count, offset int) ([]*User, error) {
	if count == 0 {
		count = searchCount
	}

	query := q.ToBson()
	log.Println(query)
	u := []*User{}

	return u, db.users.Find(query).Skip(offset).Limit(count).All(&u)
}

func (db *DB) UpdateAllStatuses() (*mgo.ChangeInfo, error) {
	// change := mgo.Change{Update: bson.M{"$set": bson.M{"online": false}}}
	t := time.Now().Add(-db.offlineTimeout)
	query := bson.M{"online": true, "lastaction": bson.M{"$lte": t}}
	return db.users.UpdateAll(query, bson.M{"$set": bson.M{"online": false}})
}

func (db *DB) ConfirmEmail(id bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$set": bson.M{"email_confirmed": true}})
}

func (db *DB) ConfirmPhone(id bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$set": bson.M{"phone_confirmed": true}})
}
