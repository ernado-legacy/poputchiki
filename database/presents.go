package database

import (
	"errors"
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

var (
	ErrBadPresentType = errors.New("Bad present type")
)

func (db *DB) AddPresent(present *models.Present) error {
	present.Id = bson.NewObjectId()
	return db.presents.Insert(present)
}

func (db *DB) RemovePresent(id bson.ObjectId) error {
	return db.presents.RemoveId(id)
}

func (db *DB) GetPresentByType(t string) (*models.Present, error) {
	present := new(models.Present)
	return present, db.presents.Find(bson.M{"title": t}).One(present)
}

func (db *DB) GetPresent(id bson.ObjectId) (*models.Present, error) {
	present := new(models.Present)
	return present, db.presents.FindId(id).One(present)
}

func (db *DB) GetAllPresents() ([]*models.Present, error) {
	presents := []*models.Present{}
	return presents, db.presents.Find(nil).Sort("-time").All(&presents)
}

func (db *DB) GetUserPresents(destination bson.ObjectId) ([]*models.PresentEvent, error) {
	presents := []*models.PresentEvent{}
	return presents, db.presentEvents.Find(bson.M{"destination": destination}).All(&presents)
}

func (db *DB) SendPresent(origin, destination bson.ObjectId, t string) (*models.PresentEvent, error) {
	p, err := db.GetPresentByType(t)
	if err == mgo.ErrNotFound {
		return nil, ErrBadPresentType
	}
	if err != nil {
		return nil, err
	}
	present := new(models.PresentEvent)
	present.Id = bson.NewObjectId()
	present.Time = time.Now()
	present.Origin = origin
	present.Destination = destination
	present.Type = p.Title
	return present, db.presentEvents.Insert(present)
}

func (db *DB) UpdatePresent(id bson.ObjectId, present *models.Present) (*models.Present, error) {
	p, err := db.GetPresent(id)
	if err != nil {
		return nil, err
	}
	p.Title = present.Title
	p.Cost = present.Cost
	if len(present.Image) > 0 {
		p.Image = present.Image
	}
	_, err = db.presents.UpsertId(id, p)
	if err != nil {
		return nil, err
	}
	return p, err
}
