package database

import (
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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
