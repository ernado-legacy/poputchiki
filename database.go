package main

import (
	"errors"
	"github.com/ernado/gotok"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"reflect"
	"strings"
	"time"
)

const (
	STRIPE_COUNT = 20
	SEARCH_COUNT = 40
)

type DB struct {
	users      *mgo.Collection
	guests     *mgo.Collection
	messages   *mgo.Collection
	statuses   *mgo.Collection
	photo      *mgo.Collection
	albums     *mgo.Collection
	files      *mgo.Collection
	video      *mgo.Collection
	audio      *mgo.Collection
	stripe     *mgo.Collection
	conftokens *mgo.Collection
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

func (db *DB) AddStatus(u bson.ObjectId, text string) (*StatusUpdate, error) {
	p := &StatusUpdate{}
	p.Id = bson.NewObjectId()
	p.Text = text
	p.Time = time.Now()
	p.User = u

	if err := db.statuses.Insert(p); err != nil {
		return nil, err
	}

	update := mgo.Change{Update: bson.M{"$set": bson.M{"statusupdate": time.Now()}}}
	_, err := db.users.FindId(u).Apply(update, &User{})

	return p, err
}

func (db *DB) UpdateStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) (*StatusUpdate, error) {
	s := &StatusUpdate{}
	change := mgo.Change{Update: bson.M{"$set": bson.M{"text": text}}}
	query := bson.M{"_id": id, "user": user}
	_, err := db.statuses.Find(query).Apply(change, s)
	s.Text = text
	return s, err
}

func (db *DB) GetStatus(id bson.ObjectId) (status *StatusUpdate, err error) {
	status = &StatusUpdate{}
	err = db.statuses.FindId(id).One(status)
	return status, err
}

func (db *DB) GetCurrentStatus(user bson.ObjectId) (status *StatusUpdate, err error) {
	status = &StatusUpdate{}
	err = db.statuses.Find(bson.M{"user": user}).Sort("-time").Limit(1).One(status)
	return status, err
}

func (db *DB) GetLastStatuses(count int) (status []*StatusUpdate, err error) {
	status = []*StatusUpdate{}
	err = db.statuses.Find(nil).Sort("-time").Limit(count).All(&status)
	return status, err
}

func (db *DB) RemoveStatusSecure(user bson.ObjectId, id bson.ObjectId) error {
	query := bson.M{"_id": id, "user": user}
	err := db.statuses.Remove(query)
	return err
}

func (db *DB) AddPhoto(user bson.ObjectId, imageJpeg File, imageWebp File, thumbnailJpeg File, thumbnailWebp File, desctiption string) (*Photo, error) {
	// creating photo
	p := &Photo{Id: bson.NewObjectId(), User: user, ImageJpeg: imageJpeg.Fid, ImageWebp: imageWebp.Fid,
		Time: time.Now(), Description: desctiption, ThumbnailJpeg: thumbnailJpeg.Fid, ThumbnailWebp: thumbnailWebp.Fid}
	err := db.photo.Insert(p)

	if err != nil {
		return nil, err
	}

	return p, err
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}

func (db *DB) AddVideo(video *Video) (*Video, error) {
	return video, db.video.Insert(video)
}

func (db *DB) AddAudio(audio *Audio) (*Audio, error) {
	return audio, db.audio.Insert(audio)
}

func (db *DB) GetAudio(id bson.ObjectId) *Audio {
	a := &Audio{}
	if db.audio.FindId(id).One(a) != nil {
		return nil
	}
	return a
}

func (db *DB) GetVideo(id bson.ObjectId) *Video {
	v := &Video{}
	if db.video.FindId(id).Select(bson.M{"liked_users": 0}).One(v) != nil {
		return nil
	}
	return v
}

func (db *DB) GetPhoto(photo bson.ObjectId) (*Photo, error) {
	p := &Photo{}
	return p, db.photo.FindId(photo).Select(bson.M{"liked_users": 0}).One(p)
}

func (db *DB) GetUserPhoto(user bson.ObjectId) ([]*Photo, error) {
	p := []*Photo{}
	err := db.photo.Find(bson.M{"user": user}).All(p)
	if err != nil {
		return nil, err
	}
	return p, err
}

func (db *DB) RemovePhoto(user bson.ObjectId, id bson.ObjectId) error {
	return db.photo.Remove(bson.M{"_id": id, "user": user})
}

// Updates photo description and return updated object; TODO: test; TODO: check photo.id != id
func (db *DB) UpdatePhoto(user, id bson.ObjectId, photo *Photo) (*Photo, error) {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"description": photo.Description}}}
	p := &Photo{}
	_, err := db.photo.Find(bson.M{"_id": id, "user": user}).Apply(change, p)
	p.Description = photo.Description
	return p, err
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
		count = STRIPE_COUNT
	}
	return s, db.stripe.Find(nil).Sort("-time").Skip(offset).Limit(count).All(&s)
}

func (db *DB) Search(q *SearchQuery, count, offset int) ([]*User, error) {
	if count == 0 {
		count = SEARCH_COUNT
	}

	query := q.ToBson()
	log.Println(query)
	u := []*User{}

	return u, db.users.Find(query).Skip(offset).Limit(count).All(&u)
}

func (db *DB) SearchStatuses(q *SearchQuery, count, offset int) ([]*StatusUpdate, error) {
	if count == 0 {
		count = SEARCH_COUNT
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

func (db *DB) SearchPhoto(q *SearchQuery, count, offset int) ([]*Photo, error) {
	if count == 0 {
		count = SEARCH_COUNT
	}

	photos := []*Photo{}
	query := q.ToBson()
	log.Println("query:", query)
	u := []*User{}
	if err := db.users.Find(query).Skip(offset).Limit(count).All(&u); err != nil {
		return photos, err
	}
	users := make([]bson.ObjectId, len(u))
	for i, user := range u {
		users[i] = user.Id
	}

	if err := db.photo.Find(bson.M{"user": bson.M{"$in": users}}).All(&photos); err != nil {
		return photos, err
	}
	if len(photos) != len(users) {
		return photos, errors.New("unexpected length")
	}

	return photos, nil
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
	t := time.Now().Add(-OfflineTimeout)
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

func (db *DB) AddLikeVideo(user bson.ObjectId, target bson.ObjectId) error {
	return db.AddLike(db.video, user, target)
}

func (db *DB) RemoveLikeVideo(user bson.ObjectId, target bson.ObjectId) error {
	return db.RemoveLike(db.video, user, target)
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

func (db *DB) GetLikesVideo(id bson.ObjectId) []*User {
	return db.GetLikes(db.video, id)
}

// {ObjectIdHex("53ac5de136c4536687000007") ObjectIdHex("53a5932a36c4531911000002") 13,09c5e0640517 10,09c64a9a818f http://msk1.cydev.ru:8080/13,09c5e0640517.webm 10,09c739840627 12,09c8f5df8949 http://msk1.cydev.ru:8080/10,09c739840627.webp  2014-06-26 21:52:33.085479286 +0400 MSK 0 35}
