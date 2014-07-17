package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"labix.org/v2/mgo/bson"
	"testing"
)

func TestUsers(t *testing.T) {
	db := TestDatabase()

	Convey("Confirmation", t, func() {
		Reset(func() {
			db.Drop()
		})
		id := bson.NewObjectId()
		u := &models.User{Id: id}
		Convey("Add", func() {
			So(db.Add(u), ShouldBeNil)
			Convey("Add guest", func() {
				guestId := bson.NewObjectId()
				guest := &models.User{Id: guestId}
				So(db.Add(guest), ShouldBeNil)
				So(db.AddGuest(id, guestId), ShouldBeNil)
				Convey("In guests", func() {
					guests, err := db.GetAllGuestUsers(id)
					So(err, ShouldBeNil)
					found := false
					for _, v := range guests {
						if v.Id == guestId {
							found = true
						}
					}
					So(found, ShouldBeTrue)
				})
			})
		})
	})

}
