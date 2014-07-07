package database

import (
	"errors"
	. "github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

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

func (db *DB) SearchPhoto(q *SearchQuery, count, offset int) ([]*Photo, error) {
	if count == 0 {
		count = searchCount
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
