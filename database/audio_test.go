package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestAudio(t *testing.T) {
	db := TestDatabase()

	Convey("Audio", t, func() {
		Reset(func() {
			db.Drop()
		})

		Convey("Add", func() {
			a := &models.Audio{}
			a.Id = bson.NewObjectId()
			a.User = bson.NewObjectId()
			u := &models.User{}
			u.Id = a.User
			So(db.Add(u), ShouldBeNil)
			_, err := db.AddAudio(a)
			So(err, ShouldBeNil)
			Convey("Get", func() {
				b := db.GetAudio(a.Id)
				So(b.User, ShouldEqual, a.User)
			})
		})
	})

}
