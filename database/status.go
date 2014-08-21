package database

import (
	// "errors"
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"sort"
	"time"
)

// AddStatus adds new status to database with provided text and id
func (db *DB) AddStatus(u bson.ObjectId, text string) (*models.Status, error) {
	p := &models.Status{}
	p.Id = bson.NewObjectId()
	p.Text = text
	p.Time = time.Now()
	p.User = u

	if err := db.statuses.Insert(p); err != nil {
		return nil, err
	}

	update := mgo.Change{Update: bson.M{"$set": bson.M{"statusupdate": p.Time, "status": text}}}
	_, err := db.users.FindId(u).Apply(update, &models.User{})

	return p, err
}

// UpdateStatusSecure updates status ensuring ownership
func (db *DB) UpdateStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) (*models.Status, error) {
	s := &models.Status{}
	change := mgo.Change{Update: bson.M{"$set": bson.M{"text": text}}}
	query := bson.M{"_id": id, "user": user}
	_, err := db.statuses.Find(query).Apply(change, s)
	if err != nil {
		return nil, err
	}
	s.Text = text

	// sync user status
	update := bson.M{"$set": bson.M{"status": text}}
	err = db.users.UpdateId(user, update)
	return s, err
}

func (db *DB) GetStatus(id bson.ObjectId) (status *models.Status, err error) {
	status = &models.Status{}
	err = db.statuses.FindId(id).One(status)
	return status, err
}

// GetCurrentStatus returs current status of user with provided id
func (db *DB) GetCurrentStatus(user bson.ObjectId) (status *models.Status, err error) {
	status = &models.Status{}
	err = db.statuses.Find(bson.M{"user": user}).Sort("-time").Limit(1).One(status)
	return status, err
}

// GetLastStatuses returns global most auctual statuses
func (db *DB) GetLastStatuses(count int) (status []*models.Status, err error) {
	status = []*models.Status{}
	err = db.statuses.Find(nil).Sort("-time").Limit(count).All(&status)
	return status, err
}

// RemoveStatusSecure removes status ensuring ownership
func (db *DB) RemoveStatusSecure(user bson.ObjectId, id bson.ObjectId) error {
	query := bson.M{"_id": id, "user": user}
	err := db.statuses.Remove(query)
	return err
}

type StatusByTime []*models.Status

func (a StatusByTime) Len() int {
	return len(a)
}

func (a StatusByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a StatusByTime) Less(i, j int) bool {
	return a[j].Time.Before(a[i].Time)
}

func (db *DB) SearchStatuses(q *models.SearchQuery, count, offset int) ([]*models.Status, error) {
	if count == 0 {
		count = searchCount
	}

	statuses := []*models.Status{}
	query := q.ToBson()
	u := []*models.User{}
	query["statusupdate"] = bson.M{"$exists": true}
	log.Println(query)
	if err := db.users.Find(query).Sort("-statusupdate").Skip(offset).Limit(count).All(&u); err != nil {
		return statuses, err
	}
	log.Println(u)
	userIds := make([]bson.ObjectId, len(u))
	users := make(map[bson.ObjectId]*models.User)
	for i, user := range u {
		users[user.Id] = user
		userIds[i] = user.Id
	}

	if err := db.statuses.Find(bson.M{"user": bson.M{"$in": userIds}}).All(&statuses); err != nil {
		return statuses, err
	}

	for i, status := range statuses {
		statuses[i].UserObject = users[status.User]
	}

	sort.Sort(StatusByTime(statuses))
	return statuses, nil
}

func (db *DB) GetTopStatuses(count, offset int) (statuses []*models.Status, err error) {
	var userIds []bson.ObjectId
	statuses = []*models.Status{}
	users := []*models.User{}
	userMap := make(map[bson.ObjectId]*models.User)

	if err = db.statuses.Find(nil).Sort("-likes").Skip(offset).Limit(count).All(&statuses); err != nil {
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
		status.UserObject = userMap[status.User]
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

func (db *DB) GetLastDayStatusesAmount(id bson.ObjectId) (int, error) {
	from := time.Now().AddDate(0, 0, -1)
	return db.statuses.Find(bson.M{"user": id, "time": bson.M{"$gte": from}}).Count()
}
