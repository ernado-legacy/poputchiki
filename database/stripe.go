package database

import (
	"github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo/bson"
	"reflect"
	"strings"
)

func (db *DB) AddStripeItem(user bson.ObjectId, media interface{}) (*models.StripeItem, error) {
	i := &models.StripeItem{}
	i.Id = bson.NewObjectId()
	i.User = user
	i.Media = media
	i.Type = strings.ToLower(reflect.TypeOf(media).Name())
	return i, db.stripe.Insert(i)
}

func (db *DB) GetStripeItem(id bson.ObjectId) (*models.StripeItem, error) {
	s := &models.StripeItem{}
	return s, db.stripe.FindId(id).One(s)
}

func (db *DB) GetStripe(count, offset int) ([]*models.StripeItem, error) {
	s := []*models.StripeItem{}
	if count == 0 {
		count = stripeCount
	}
	return s, db.stripe.Find(nil).Sort("-time").Skip(offset).Limit(count).All(&s)
}
