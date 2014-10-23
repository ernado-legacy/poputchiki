package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func (db *DB) AddAdvertisement(i *models.StripeItem, media interface{}) (*models.StripeItem, error) {
	i.Media = media
	if len(i.Id.Hex()) == 0 {
		i.Id = bson.NewObjectId()
	}
	if media == nil {
		i.Type = "text"
	} else {
		i.Type = "photo"
	}
	i.Time = time.Now()
	return i, db.advertisements.Insert(i)
}

func (db *DB) GetAdvertisment(id bson.ObjectId) (*models.StripeItem, error) {
	s := &models.StripeItem{}
	return s, db.advertisements.FindId(id).One(s)
}

func (db *DB) RemoveAdvertisment(user, id bson.ObjectId) error {
	return db.advertisements.Remove(bson.M{"user": user, "_id": id})
}

func (db *DB) GetAds(count, offset int) (models.Stripe, error) {
	s := []*models.StripeItem{}
	if count == 0 {
		count = stripeCount
	}
	return models.Stripe(s), db.advertisements.Find(nil).Sort("-time").Skip(offset).Limit(count).All(&s)
}
