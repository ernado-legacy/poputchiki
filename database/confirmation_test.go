package database

import (
	// "github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"labix.org/v2/mgo/bson"
	"testing"
)

func TestConfirmation(t *testing.T) {
	db := TestDatabase()

	Convey("Confirmation", t, func() {
		Reset(func() {
			db.Drop()
		})
		id := bson.NewObjectId()
		Convey("Add", func() {
			t := db.NewConfirmationToken(id)
			So(t, ShouldNotBeNil)
			Convey("Get", func() {
				t2 := db.GetConfirmationToken(t.Token)
				So(t2.Id, ShouldEqual, t.Id)
				Convey("Removed", func() {
					t3 := db.GetConfirmationToken(t.Token)
					So(t3, ShouldBeNil)
				})
			})
		})
	})

}
