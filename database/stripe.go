package database

import (
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"strings"
	"time"
)

func (db *DB) AddStripeItem(i *models.StripeItem, media interface{}) (*models.StripeItem, error) {
	i.Media = media
	if i.Type == "" {
		i.Type = strings.ToLower(reflect.TypeOf(media).Name())
	}
	i.Time = time.Now()
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
