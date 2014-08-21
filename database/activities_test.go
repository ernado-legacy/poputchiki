package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestActivities(t *testing.T) {
	db := TestDatabase()

	Convey("Confirmation", t, func() {
		Reset(func() {
			db.Drop()
		})
		id := bson.NewObjectId()
		u := &models.User{Id: id}
		Convey("Add user", func() {
			So(db.Add(u), ShouldBeNil)
		})
	})

}
