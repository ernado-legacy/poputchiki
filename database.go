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
	statuses *mgo.Collection
	photo    *mgo.Collection
	albums   *mgo.Collection
	files    *mgo.Collection
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
	err = db.messages.Find(bson.M{"user": userReciever, "origin": userOrigin}).All(&messages)
	return messages, err
}

func (db *DB) AddStatus(u bson.ObjectId, text string) (*StatusUpdate, error) {
	p := StatusUpdate{}
	p.Id = bson.NewObjectId()
	p.Text = text
	p.Time = time.Now()
	p.User = u

	return &p, db.statuses.Insert(&p)
}

func (db *DB) AddCommentToStatus(user bson.ObjectId, status bson.ObjectId, text string) (*Comment, error) {
	c := &Comment{bson.NewObjectId(), user, text, time.Now()}
	change := mgo.Change{Update: bson.M{"$addToSet": bson.M{"comments": c}}}
	u := &StatusUpdate{}
	_, err := db.statuses.FindId(status).Apply(change, u)
	return c, err
}

func (db *DB) RemoveCommentFromStatusSecure(user bson.ObjectId, id bson.ObjectId) error {
	change := mgo.Change{Update: bson.M{"$pull": bson.M{"comments": bson.M{"_id": id}}}}
	query := bson.M{"comments._id": id, "user": user}
	u := &StatusUpdate{}
	_, err := db.statuses.Find(query).Apply(change, u)
	return err
}

func (db *DB) UpdateCommentToStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) error {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"comments.$.text": text}}}
	query := bson.M{"comments._id": id, "user": user}
	u := &StatusUpdate{}
	_, err := db.statuses.Find(query).Apply(change, u)
	return err
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

func (db *DB) AddAlbum(user bson.ObjectId, album *Album) (*Album, error) {
	album.User = user
	album.Time = time.Now()
	album.Id = bson.NewObjectId()
	return album, db.albums.Insert(album)
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}

func (db *DB) AddPhotoToAlbum(user bson.ObjectId, album bson.ObjectId, photo bson.ObjectId) error {
	change := mgo.Change{Update: bson.M{"$addToSet": bson.M{"photo": photo}}}
	_, err := db.albums.Find(bson.M{"_id": album, "user": user}).Apply(change, &Album{})

	return err
}

func (db *DB) AddCommentToPhoto(user bson.ObjectId, photo bson.ObjectId, c *Comment) error {
	c.User = user
	c.Id = bson.NewObjectId()
	c.Time = time.Now()
	change := mgo.Change{Update: bson.M{"$addToSet": bson.M{"comments": c}}}
	p := &Photo{}
	_, err := db.photo.FindId(photo).Apply(change, p)
	return err
}

func (db *DB) GetPhoto(photo bson.ObjectId) (*Photo, error) {
	p := &Photo{}
	return p, db.photo.FindId(photo).One(p)
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
