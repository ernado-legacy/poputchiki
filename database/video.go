package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
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
	if db.video.FindId(id).One(v) != nil {
		return nil
	}
	return v
}

func (db *DB) RemoveVideo(user bson.ObjectId, id bson.ObjectId) error {
	return db.video.Remove(bson.M{"_id": id, "user": user})
}

func (db *DB) GetUserVideo(id bson.ObjectId) ([]*models.Video, error) {
	v := []*models.Video{}
	return v, db.video.Find(bson.M{"user": id}).Sort("-time").All(&v)
}

func (db *DB) AddVideo(video *models.Video) (*models.Video, error) {
	return video, db.video.Insert(video)
}

func (db *DB) UpdateVideoWebm(id bson.ObjectId, fid string) error {
	return db.video.UpdateId(id, bson.M{"$set": bson.M{"video_webm": fid}})
}

func (db *DB) UpdateVideoMpeg(id bson.ObjectId, fid string) error {
	return db.video.UpdateId(id, bson.M{"$set": bson.M{"video_mpeg": fid}})
}

func (db *DB) UpdateVideoThumbnails(id bson.ObjectId, tjpeg, twebp string) error {
	return db.video.UpdateId(id, bson.M{"$set": bson.M{"thumbnail_webp": twebp, "thumbnail_jpeg": tjpeg}})
}
