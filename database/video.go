package database

import (
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo/bson"
)

func (db *DB) GetLikesVideo(id bson.ObjectId) []*models.User {
	return db.GetLikes(db.video, id)
}

func (db *DB) RemoveLikeVideo(user bson.ObjectId, target bson.ObjectId) error {
	return db.RemoveLike(db.video, user, target)
}

func (db *DB) AddLikeVideo(user bson.ObjectId, target bson.ObjectId) error {
	return db.AddLike(db.video, user, target)
}

func (db *DB) GetVideo(id bson.ObjectId) *models.Video {
	v := &models.Video{}
	if db.video.FindId(id).Select(bson.M{"liked_users": 0}).One(v) != nil {
		return nil
	}
	return v
}

func (db *DB) AddVideo(video *models.Video) (*models.Video, error) {
	return video, db.video.Insert(video)
}
