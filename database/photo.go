package database

import (
	"time"

	. "github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// AddPhoto add new photo to database with provided image and thumbnail
func (db *DB) AddPhoto(user bson.ObjectId, image, thumbnail string) (*Photo, error) {
	// creating photo
	p := &Photo{Id: bson.NewObjectId(), User: user, ImageJpeg: image,
		Time: time.Now(), ThumbnailJpeg: thumbnail}
	err := db.photo.Insert(p)

	if err != nil {
		return nil, err
	}

	return p, err
}

// AddPhotoHidden adds new hidden photo to database with provided image and thumbnail
func (db *DB) AddPhotoHidden(user bson.ObjectId, image, thumbnail string) (*Photo, error) {
	// creating photo
	p := &Photo{Id: bson.NewObjectId(), User: user, ImageJpeg: image,
		Time: time.Now(), ThumbnailJpeg: thumbnail, Hidden: true}
	err := db.photo.Insert(p)

	if err != nil {
		return nil, err
	}

	return p, err
}

// GetPhoto returs photo by id
func (db *DB) GetPhoto(photo bson.ObjectId) (*Photo, error) {
	p := &Photo{}
	return p, db.photo.FindId(photo).One(p)
}

// GetUserPhoto returns avatar photo of user with provided user id
func (db *DB) GetUserPhoto(user bson.ObjectId) ([]*Photo, error) {
	p := []*Photo{}
	err := db.photo.Find(bson.M{"user": user, "hidden": bson.M{"$ne": true}}).Sort("-time").All(&p)
	if err != nil {
		return nil, err
	}
	return p, err
}

// RemovePhoto removes photo from database ensuring ownership of user
func (db *DB) RemovePhoto(user bson.ObjectId, id bson.ObjectId) error {
	_, err := db.users.UpdateAll(bson.M{"_id": user, "avatar": id}, bson.M{"$unset": bson.M{"avatar": ""}})
	if err != nil {
		return err
	}
	return db.photo.Remove(bson.M{"_id": id, "user": user})
}

// Updates photo description and return updated object
func (db *DB) UpdatePhoto(user, id bson.ObjectId, photo *Photo) (*Photo, error) {
	change := mgo.Change{Update: bson.M{"$set": bson.M{"description": photo.Description}}}
	p := &Photo{}
	_, err := db.photo.Find(bson.M{"_id": id, "user": user}).Apply(change, p)
	p.Description = photo.Description
	return p, err
}

func (db *DB) SetPhotoHidden(user, id bson.ObjectId, hidden bool) error {
	update := bson.M{"$set": bson.M{"hidden": hidden}}
	selector := bson.M{"_id": id, "user": user}
	// _, err :=
	return db.photo.Update(selector, update)
}

func (db *DB) SearchAllPhoto(pagination Pagination) ([]*Photo, int, error) {
	if pagination.Count == 0 {
		pagination.Count = searchCount
	}
	users := []bson.ObjectId{}
	u := []*User{}
	count, _ := db.photo.Count()
	sorting := "-time"
	photos := []*Photo{}
	err := db.photo.Find(bson.M{"hidden": bson.M{"$ne": true}}).Sort(sorting).Skip(pagination.Offset).Limit(pagination.Count).All(&photos)
	if err != nil {
		return photos, count, err
	}
	for _, p := range photos {
		users = append(users, p.User)
	}
	query := bson.M{"_id": bson.M{"$in": users}}
	if err := db.users.Find(query).All(&u); err != nil {
		return photos, count, err
	}
	idToUser := make(map[bson.ObjectId]*User)
	for _, user := range u {
		idToUser[user.Id] = user
	}

	for _, p := range photos {
		p.UserObject = idToUser[p.User]
	}
	return photos, count, err
}

// SearchPhoto returns photo filtered by query and adjusted by count and offset
func (db *DB) SearchPhoto(q *SearchQuery, pagination Pagination) ([]*Photo, error) {
	if pagination.Count == 0 {
		pagination.Count = searchCount
	}

	photos := []*Photo{}
	query := q.ToBson()

	u := []*User{}
	if err := db.users.Find(query).Sort("-rating").Skip(pagination.Offset).Limit(pagination.Count).All(&u); err != nil {
		return photos, err
	}
	users := make([]bson.ObjectId, len(u))
	usersMap := make(map[bson.ObjectId]*User)
	for i, user := range u {
		users[i] = user.Id
		usersMap[user.Id] = user
	}
	if err := db.photo.Find(bson.M{"user": bson.M{"$in": users}, "hidden": bson.M{"$ne": true}}).Sort("-time").All(&photos); err != nil {
		return photos, err
	}
	for _, photo := range photos {
		photo.UserObject = usersMap[photo.User]
	}
	return photos, nil
}

// SearchPhoto returns photo filtered by query and adjusted by count and offset
func (db *DB) SearchMedia(q *SearchQuery, pagination Pagination) ([]*Photo, error) {
	if pagination.Count == 0 {
		pagination.Count = searchCount
	}

	photos := []*Photo{}
	query := q.ToBson()
	u := []*User{}
	if err := db.users.Find(query).Sort("-rating").Skip(pagination.Offset).Limit(pagination.Count).All(&u); err != nil {
		return photos, err
	}
	users := make([]bson.ObjectId, len(u))
	usersMap := make(map[bson.ObjectId]*User)
	for i, user := range u {
		users[i] = user.Id
		usersMap[user.Id] = user
	}
	if err := db.photo.Find(bson.M{"user": bson.M{"$in": users}, "hidden": bson.M{"$ne": true}}).Sort("-time").All(&photos); err != nil {
		return photos, err
	}
	for _, photo := range photos {
		photo.UserObject = usersMap[photo.User]
	}
	return photos, nil
}

func (db *DB) AddLikePhoto(user bson.ObjectId, target bson.ObjectId) error {
	return db.AddLike(db.photo, user, target)
}

func (db *DB) RemoveLikePhoto(user bson.ObjectId, target bson.ObjectId) error {
	return db.RemoveLike(db.photo, user, target)
}

func (db *DB) GetLikesPhoto(id bson.ObjectId) []*User {
	return db.GetLikes(db.photo, id)
}
