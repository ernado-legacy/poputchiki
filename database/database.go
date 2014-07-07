package database

import (
	"errors"
	"github.com/ernado/gotok"
	. "github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"reflect"
	"strings"
	"time"
)

const (
	stripeCount = 20
	searchCount = 40
)

var (
	collection           = "users"
	citiesCollection     = "cities"
	countriesCollection  = "countries"
	guestsCollection     = "guests"
	messagesCollection   = "messages"
	statusesCollection   = "statuses"
	photoCollection      = "photo"
	albumsCollection     = "albums"
	filesCollection      = "files"
	videoCollection      = "audio"
	conftokensCollection = "conftokens"
	audioCollection      = "video"
	stripeCollection     = "stripe"
	tokenCollection      = "tokens"
)

type DB struct {
	users          *mgo.Collection
	guests         *mgo.Collection
	messages       *mgo.Collection
	statuses       *mgo.Collection
	photo          *mgo.Collection
	albums         *mgo.Collection
	files          *mgo.Collection
	video          *mgo.Collection
	audio          *mgo.Collection
	stripe         *mgo.Collection
	conftokens     *mgo.Collection
	salt           string
	offlineTimeout time.Duration
}

func New(name, salt string, timeout time.Duration, session *mgo.Session) *DB {
	db := session.DB(name)
	coll := db.C(collection)
	gcoll := db.C(guestsCollection)
	mcoll := db.C(messagesCollection)
	scoll := db.C(statusesCollection)
	pcoll := db.C(photoCollection)
	acoll := db.C(albumsCollection)
	fcoll := db.C(filesCollection)
	vcoll := db.C(videoCollection)
	aucoll := db.C(audioCollection)
	stcoll := db.C(stripeCollection)
	ctcoll := db.C(conftokensCollection)
	return &DB{coll, gcoll, mcoll, scoll, pcoll, acoll, fcoll, vcoll, aucoll, stcoll, ctcoll, salt, timeout}
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

func (db *DB) Salt() string {
	return db.salt
}

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

func (db *DB) AddGuest(id bson.ObjectId, guest bson.ObjectId) error {
	g := Guest{bson.NewObjectId(), id, guest, time.Now()}
	_, err := db.guests.Upsert(bson.M{"user": id, "guest": guest}, &g)
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

func (db *DB) AddMessage(m *Message) error {
	return db.messages.Insert(m)
}

func (db *DB) RemoveMessage(id bson.ObjectId) error {
	return db.messages.RemoveId(id)
}

func (db *DB) GetMessage(id bson.ObjectId) (*Message, error) {
	message := Message{}
	err := db.messages.FindId(id).One(&message)
	return &message, err
}

func (db *DB) GetChats(id bson.ObjectId) ([]*User, error) {
	var users []*User
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

func (db *DB) GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) (messages []*Message, err error) {
	err = db.messages.Find(bson.M{"user": userReciever, "chat": userOrigin}).All(&messages)
	return messages, err
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}

func (db *DB) AddStripeItem(user bson.ObjectId, media interface{}) (*StripeItem, error) {
	i := &StripeItem{}
	i.Id = bson.NewObjectId()
	i.User = user
	i.Media = media
	i.Type = strings.ToLower(reflect.TypeOf(media).Name())
	return i, db.stripe.Insert(i)
}

func (db *DB) GetStripeItem(id bson.ObjectId) (*StripeItem, error) {
	s := &StripeItem{}
	return s, db.stripe.FindId(id).One(s)
}

func (db *DB) GetStripe(count, offset int) ([]*StripeItem, error) {
	s := []*StripeItem{}
	if count == 0 {
		count = stripeCount
	}
	return s, db.stripe.Find(nil).Sort("-time").Skip(offset).Limit(count).All(&s)
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

func (db *DB) SearchStatuses(q *SearchQuery, count, offset int) ([]*StatusUpdate, error) {
	if count == 0 {
		count = searchCount
	}

	statuses := []*StatusUpdate{}
	query := q.ToBson()
	u := []*User{}
	query["statusupdate"] = bson.M{"$exists": true}
	if err := db.users.Find(query).Sort("-statusupdate").Skip(offset).Limit(count).All(&u); err != nil {
		return statuses, err
	}
	users := make([]bson.ObjectId, len(u))
	for i, user := range u {
		users[i] = user.Id
	}

	if err := db.statuses.Find(bson.M{"user": bson.M{"$in": users}}).All(&statuses); err != nil {
		return statuses, err
	}
	if len(statuses) != len(users) {
		return statuses, errors.New("unexpected length")
	}

	for i, user := range u {
		statuses[i].ImageJpeg = user.AvatarJpeg
		statuses[i].ImageWebp = user.AvatarWebp
		statuses[i].Name = user.Name
	}

	return statuses, nil
}

func (db *DB) NewConfirmationTokenValue(id bson.ObjectId, token string) *EmailConfirmationToken {
	t := &EmailConfirmationToken{}
	t.Id = bson.NewObjectId()
	t.User = id
	t.Time = time.Now()
	t.Token = token
	err := db.conftokens.Insert(t)
	log.Println("[NewConfirmationToken]", "inserted", t.Token)
	if err != nil {
		log.Println("[NewConfirmationToken]", err)
		return nil
	}
	return t
}

func (db *DB) NewConfirmationToken(id bson.ObjectId) *EmailConfirmationToken {
	return db.NewConfirmationTokenValue(id, gotok.Generate(id).Token)
}

func (db *DB) GetConfirmationToken(token string) *EmailConfirmationToken {
	t := &EmailConfirmationToken{}
	selector := bson.M{"token": token}
	log.Println("[GetConfirmationToken]", "searching", token)
	if err := db.conftokens.Find(selector).One(t); err != nil {
		log.Println("[GetConfirmationToken]", err)
		return nil
	}

	if err := db.conftokens.Remove(selector); err != nil {
		log.Println("[GetConfirmationToken]", "[remove]", err)
		return nil
	}
	return t
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

func (db *DB) AddLike(coll *mgo.Collection, user bson.ObjectId, target bson.ObjectId) error {
	if err := coll.UpdateId(target, bson.M{"$push": bson.M{"liked_users": user}}); err != nil {
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

func (db *DB) AddLikePhoto(user bson.ObjectId, target bson.ObjectId) error {
	return db.AddLike(db.photo, user, target)
}

func (db *DB) RemoveLikePhoto(user bson.ObjectId, target bson.ObjectId) error {
	return db.RemoveLike(db.photo, user, target)
}

func (db *DB) GetLikes(coll *mgo.Collection, id bson.ObjectId) []*User {
	var likersID []bson.ObjectId
	var likers []*User

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

func (db *DB) GetLikesPhoto(id bson.ObjectId) []*User {
	return db.GetLikes(db.photo, id)
}
