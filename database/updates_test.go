package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestUpdates(t *testing.T) {
	db := TestDatabase()
	Convey("Confirmation tokens", t, func() {
		Reset(db.Drop)
		origin := bson.NewObjectId()
		destination := bson.NewObjectId()
		Convey("Add users", func() {
			uOrigin := &models.User{Id: origin, Name: "Alex"}
			uDestination := &models.User{Id: destination, Name: "Petya"}
			So(db.Add(uOrigin), ShouldBeNil)
			So(db.Add(uDestination), ShouldBeNil)
			Convey("Integrity", Integrity(db, uOrigin))
			Convey("Integrity", Integrity(db, uDestination))
			Convey("Add update", func() {
				t := models.UpdateGuests
				u, err := db.AddUpdate(destination, origin, t, nil)
				So(err, ShouldBeNil)
				So(u, ShouldNotBeNil)
				Convey("Integrity", Integrity(db, uOrigin))
				Convey("Integrity", Integrity(db, uDestination))
				Convey("Get updates", func() {
					Convey("All", func() {
						updates, err := db.GetUpdates(destination, t, models.Pagination{})
						So(err, ShouldBeNil)
						So(len(updates), ShouldEqual, 1)
					})
					Convey("Count", func() {
						counters, err := db.GetUpdatesCount(destination)
						So(err, ShouldBeNil)
						So(len(counters), ShouldEqual, 2)
					})
					Convey("Type count", func() {
						count, err := db.GetUpdatesTypeCount(destination, t)
						So(err, ShouldBeNil)
						So(count, ShouldEqual, 1)
					})
				})
				Convey("Set read", func() {
					So(db.SetUpdateRead(destination, u.Id), ShouldBeNil)
					Convey("Integrity", Integrity(db, uOrigin))
					Convey("Integrity", Integrity(db, uDestination))
					Convey("Get updates", func() {
						Convey("All", func() {
							updates, err := db.GetUpdates(destination, t, models.Pagination{})
							So(err, ShouldBeNil)
							So(len(updates), ShouldEqual, 1)
						})
						Convey("Count", func() {
							counters, err := db.GetUpdatesCount(destination)
							So(err, ShouldBeNil)
							So(len(counters), ShouldEqual, 1)
						})
						Convey("Type count", func() {
							count, err := db.GetUpdatesTypeCount(destination, t)
							So(err, ShouldBeNil)
							So(count, ShouldEqual, 0)
						})
					})
				})
			})
		})
	})
}
