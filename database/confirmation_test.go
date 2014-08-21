package database

import (
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestConfirmation(t *testing.T) {
	db := TestDatabase()
	Convey("Confirmation tokens", t, func() {
		Reset(db.Drop)
		Convey("Add", func() {
			t := db.NewConfirmationToken(bson.NewObjectId())
			So(t, ShouldNotBeNil)
			Convey("Get", func() {
				So(db.GetConfirmationToken(t.Token).Id, ShouldEqual, t.Id)
				Convey("Removed", func() {
					So(db.GetConfirmationToken(t.Token), ShouldBeNil)
				})
			})
		})
	})

}
