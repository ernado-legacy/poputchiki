package database

import (
	"errors"
	"fmt"
	. "github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

func (db *DB) SetVip(id bson.ObjectId, vip bool) error {
	return db.users.Update(bson.M{"_id": id}, bson.M{"$set": bson.M{"vip": vip}})
}

func (db *DB) SetVipTill(id bson.ObjectId, t time.Time) error {
	return db.users.Update(bson.M{"_id": id}, bson.M{"$set": bson.M{"vip_till": t}})
}

func (db *DB) addToken(id bson.ObjectId, field, token string) error {
	value := bson.M{field: token}
	update := bson.M{"$addToSet": value}
	selector := bson.M{"_id": bson.M{"$ne": id}, field: token}
	if err := db.users.UpdateId(id, update); err != nil {
		return err
	}
	update = bson.M{"$pull": value}
	_, err := db.users.UpdateAll(selector, update)
	return err
}

func (db *DB) removeToken(id bson.ObjectId, field, token string) error {
	update := bson.M{"$pull": bson.M{field: token}}
	return db.users.UpdateId(id, update)
}

func (db *DB) RemoveAndroidToken(id bson.ObjectId, token string) error {
	return db.removeToken(id, "android_token", token)
}

func (db *DB) RemoveIosToken(id bson.ObjectId, token string) error {
	return db.removeToken(id, "ios_token", token)
}

func (db *DB) AddAndroidToken(id bson.ObjectId, token string) error {
	return db.addToken(id, "android_token", token)
}

func (db *DB) AddIosToken(id bson.ObjectId, token string) error {
	return db.addToken(id, "ios_token", token)
}

func (db *DB) Online() int {
	n, _ := db.users.Find(bson.M{"online": true}).Count()
	return n
}

func (db *DB) RegisteredCount(duration time.Duration) int {
	now := time.Now()
	from := now.Add(-duration)
	n, _ := db.users.Find(bson.M{"registered": bson.M{"$gte": from}}).Count()
	return n
}

func (db *DB) Get(id bson.ObjectId) *User {
	var u User
	err := db.users.FindId(id).One(&u)

	if err != nil {
		if err != mgo.ErrNotFound {
			log.Println("getting user by id error:", err, id)
		}
		return nil
	}

	return &u
}

// Get user by username (email)
func (db *DB) GetUsername(username string) *User {
	var u User
	err := db.users.Find(bson.M{"email": username}).One(&u)

	if err != nil {
		if err != mgo.ErrNotFound {
			log.Println("getting user by username error:", err)
		}
		return nil
	}
	return &u

}

func (db *DB) AddToFavorites(id bson.ObjectId, favId bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$addToSet": bson.M{"favorites": favId}})
}

func (db *DB) RemoveFromFavorites(id bson.ObjectId, favId bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$pull": bson.M{"favorites": favId}})
}

func (db *DB) AvatarRemove(user, id bson.ObjectId) error {
	selector := bson.M{"_id": user, "avatar": id}
	update := bson.M{"$unser": "avatar"}
	return db.users.Update(selector, update)
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

func (db *DB) GetBlacklisted(id bson.ObjectId) []*User {
	var ids []bson.ObjectId
	var blacklist []*User

	// first query - distinct favorites id's from user
	err := db.users.FindId(id).Distinct("blacklist", &ids)
	if err != nil {
		return nil
	}

	// second query - get all users with favorites id's
	// query: db.users.find({_id: {$in: favoriteIds}})
	err = db.users.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&blacklist)

	if err != nil {
		return nil
	}

	return blacklist
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
	return db.users.UpdateId(id, bson.M{"$addToSet": bson.M{"blacklist": blacklisted}})
}

func (db *DB) RemoveFromBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$pull": bson.M{"blacklist": blacklisted}})
}

type guestByTime []*GuestUser

func (a guestByTime) Len() int {
	return len(a)
}

func (a guestByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a guestByTime) Less(i, j int) bool {
	return a[j].Time.Before(a[i].Time)
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

func (db *DB) GetAllUsersWithFavorite(id bson.ObjectId) ([]*User, error) {
	var users []*User
	return users, db.users.Find(bson.M{"favorites": id}).All(&users)
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
	_, err := db.users.FindId(id).Apply(change, new(User))
	return err
}

func (db *DB) SetAvatar(user, avatar bson.ObjectId) error {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"avatar": avatar}}}
	_, err := db.users.FindId(user).Apply(change, &bson.M{})
	return err
}

func (db *DB) Search(q *SearchQuery, pagination Pagination) ([]*User, int, error) {
	if pagination.Count == 0 {
		pagination.Count = searchCount
	}

	query := q.ToBson()
	u := []*User{}

	sorting := "-rating"
	if len(q.Registered) > 0 {
		sorting = "-registered"
	}
	count, err := db.users.Find(query).Count()
	if err != nil {
		return u, 0, err
	}
	return u, count, db.users.Find(query).Sort(sorting).Skip(pagination.Offset).Limit(pagination.Count).All(&u)
}

func (db *DB) UpdateAllStatuses() (*mgo.ChangeInfo, error) {
	// change := mgo.Change{Update: bson.M{"$set": bson.M{"online": false}}}
	t := time.Now().Add(-db.offlineTimeout)
	query := bson.M{"online": true, "lastaction": bson.M{"$lte": t}}
	return db.users.UpdateAll(query, bson.M{"$set": bson.M{"online": false}})
}

func (db *DB) UpdateAllVip() (*mgo.ChangeInfo, error) {
	query := bson.M{"vip": true, "vip_till": bson.M{"$lte": time.Now()}}
	return db.users.UpdateAll(query, bson.M{"$set": bson.M{"vip": false}})
}

func (db *DB) ConfirmEmail(id bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$set": bson.M{"email_confirmed": true}})
}

func (db *DB) ConfirmPhone(id bson.ObjectId) error {
	return db.users.UpdateId(id, bson.M{"$set": bson.M{"phone_confirmed": true}})
}

func (db *DB) SetRating(id bson.ObjectId, rating float64) error {
	return db.users.UpdateId(id, bson.M{"$set": bson.M{"rating": rating}})
}

func (db *DB) ChangeRating(id bson.ObjectId, delta float64) error {
	return db.users.UpdateId(id, bson.M{"$inc": bson.M{"rating": delta}})
}

func (db *DB) DegradeRating(amount float64) (*mgo.ChangeInfo, error) {
	return db.users.UpdateAll(bson.M{"rating": bson.M{"$gt": 0}}, bson.M{"$inc": bson.M{"rating": -1 * amount}})
}

func (db *DB) NormalizeRating() (*mgo.ChangeInfo, error) {
	info := new(mgo.ChangeInfo)
	t, err := db.users.UpdateAll(bson.M{"rating": bson.M{"$gt": 100}}, bson.M{"$set": bson.M{"rating": 100.0}})
	if err != nil {
		return nil, err
	}
	info.Updated += t.Updated
	t, err = db.users.UpdateAll(bson.M{"rating": bson.M{"$lt": 0}}, bson.M{"$set": bson.M{"rating": 0.0}})
	if err != nil {
		return nil, err
	}
	info.Updated += t.Updated
	return info, nil
}

func (db *DB) UserIsSubscribed(id bson.ObjectId, subscription string) (bool, error) {
	var found bool
	for _, v := range Subscriptions {
		if v == subscription {
			found = true
		}
	}
	if !found {
		return false, errors.New(fmt.Sprintf("bad subscription <%s>", subscription))
	}
	query := bson.M{"_id": id, "subscriptions": subscription}
	n, err := db.users.Find(query).Count()
	if err == mgo.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (db *DB) GetUsersByEmail(email string) (Users, error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", email)}
	query := bson.M{"email": pattern}
	users := new(Users)
	return *users, db.users.Find(query).Limit(10).All(users)
}
