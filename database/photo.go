package database

import (
	. "github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

// AddPhoto add new photo to database with provided image and thumbnail
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

// GetPhoto returs photo by id
func (db *DB) GetPhoto(photo bson.ObjectId) (*Photo, error) {
	p := &Photo{}
	return p, db.photo.FindId(photo).Select(bson.M{"liked_users": 0}).One(p)
}

// GetUserPhoto returns avatar photo of user with provided user id
func (db *DB) GetUserPhoto(user bson.ObjectId) ([]*Photo, error) {
	p := []*Photo{}
	err := db.photo.Find(bson.M{"user": user}).Sort("-time").All(&p)
	if err != nil {
		return nil, err
	}
	return p, err
}

// RemovePhoto removes photo from database ensuring ownership of user
func (db *DB) RemovePhoto(user bson.ObjectId, id bson.ObjectId) error {
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

// SearchPhoto returns photo filtered by query and adjusted by count and offset
func (db *DB) SearchPhoto(q *SearchQuery, count, offset int) ([]*Photo, error) {
	if count == 0 {
		count = searchCount
	}

	photos := []*Photo{}
	query := q.ToBson()
	u := []*User{}
	if err := db.users.Find(query).Skip(offset).Limit(count).All(&u); err != nil {
		return photos, err
	}
	users := make([]bson.ObjectId, len(u))
	for i, user := range u {
		users[i] = user.Id
	}
	if err := db.photo.Find(bson.M{"user": bson.M{"$in": users}}).Sort("-time").All(&photos); err != nil {
		return photos, err
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
