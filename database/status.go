package database

import (
	// "errors"
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	// "log"
	"time"
)

// AddStatus adds new status to database with provided text and id
func (db *DB) AddStatus(u bson.ObjectId, text string) (*models.StatusUpdate, error) {
	p := &models.StatusUpdate{}
	p.Id = bson.NewObjectId()
	p.Text = text
	p.Time = time.Now()
	p.User = u

	if err := db.statuses.Insert(p); err != nil {
		return nil, err
	}

	update := mgo.Change{Update: bson.M{"$set": bson.M{"statusupdate": time.Now()}}}
	_, err := db.users.FindId(u).Apply(update, &models.User{})

	return p, err
}

// UpdateStatusSecure updates status ensuring ownership
func (db *DB) UpdateStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) (*models.StatusUpdate, error) {
	s := &models.StatusUpdate{}
	change := mgo.Change{Update: bson.M{"$set": bson.M{"text": text}}}
	query := bson.M{"_id": id, "user": user}
	_, err := db.statuses.Find(query).Apply(change, s)
	s.Text = text
	return s, err
}

func (db *DB) GetStatus(id bson.ObjectId) (status *models.StatusUpdate, err error) {
	status = &models.StatusUpdate{}
	err = db.statuses.FindId(id).One(status)
	return status, err
}

// GetCurrentStatus returs current status of user with provided id
func (db *DB) GetCurrentStatus(user bson.ObjectId) (status *models.StatusUpdate, err error) {
	status = &models.StatusUpdate{}
	err = db.statuses.Find(bson.M{"user": user}).Sort("-time").Limit(1).One(status)
	return status, err
}

// GetLastStatuses returns global most auctual statuses
func (db *DB) GetLastStatuses(count int) (status []*models.StatusUpdate, err error) {
	status = []*models.StatusUpdate{}
	err = db.statuses.Find(nil).Sort("-time").Limit(count).All(&status)
	return status, err
}

// RemoveStatusSecure removes status ensuring ownership
func (db *DB) RemoveStatusSecure(user bson.ObjectId, id bson.ObjectId) error {
	query := bson.M{"_id": id, "user": user}
	err := db.statuses.Remove(query)
	return err
}

func (db *DB) SearchStatuses(q *models.SearchQuery, count, offset int) ([]*models.StatusUpdate, error) {
	if count == 0 {
		count = searchCount
	}

	statuses := []*models.StatusUpdate{}
	query := q.ToBson()
	u := []*models.User{}
	query["statusupdate"] = bson.M{"$exists": true}
	if err := db.users.Find(query).Sort("-statusupdate").Skip(offset).Limit(count).All(&u); err != nil {
		return statuses, err
	}
	userIds := make([]bson.ObjectId, len(u))
	users := make(map[bson.ObjectId]*models.User)
	for i, user := range u {
		users[user.Id] = user
		userIds[i] = user.Id
	}

	if err := db.statuses.Find(bson.M{"user": bson.M{"$in": userIds}}).All(&statuses); err != nil {
		return statuses, err
	}
	// if len(statuses) != len(users) {
	// 	log.Println(statuses, users)
	// 	return statuses, errors.New("unexpected length")
	// }

	for i, status := range statuses {
		user := users[status.User]
		statuses[i].ImageJpeg = user.AvatarJpeg
		statuses[i].ImageWebp = user.AvatarWebp
		statuses[i].Name = user.Name
		statuses[i].Birthday = user.Birthday
	}

	return statuses, nil
}

func (db *DB) GetTopStatuses(count, offset int) (statuses []*models.StatusUpdate, err error) {
	var userIds []bson.ObjectId
	statuses = []*models.StatusUpdate{}
	users := []*models.User{}
	userMap := make(map[bson.ObjectId]*models.User)

	if err = db.statuses.Find(nil).Sort("-likes").Skip(offset).Limit(count).All(statuses); err != nil {
		return
	}

	for _, status := range statuses {
		userIds = append(userIds, status.User)
	}

	if err = db.users.Find(bson.M{"_id": bson.M{"$in": userIds}}).All(&users); err != nil {
		return
	}

	for _, user := range users {
		userMap[user.Id] = user
	}

	for _, status := range statuses {
		user := userMap[status.User]
		status.Name = user.Name
		status.ImageJpeg = user.AvatarJpeg
		status.ImageWebp = user.AvatarWebp
		status.Birthday = user.Birthday
	}

	return
}

func (db *DB) AddLikeStatus(user bson.ObjectId, target bson.ObjectId) error {
	return db.AddLike(db.statuses, user, target)
}

func (db *DB) RemoveLikeStatus(user bson.ObjectId, target bson.ObjectId) error {
	return db.RemoveLike(db.statuses, user, target)
}

func (db *DB) GetLikesStatus(id bson.ObjectId) []*models.User {
	return db.GetLikes(db.statuses, id)
}
