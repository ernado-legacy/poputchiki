package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestPresents(t *testing.T) {
	db := TestDatabase()
	Convey("Presents", t, func() {
		Reset(db.Drop)
		Convey("Add", func() {
			p := new(models.Present)
			p.Cost = 100
			p.Image = "test"
			p.Title = "cat"
			So(db.AddPresent(p), ShouldBeNil)
			Convey("Find by title", func() {
				p, err := db.GetPresentByType(p.Title)
				So(err, ShouldBeNil)
				So(p.Cost, ShouldEqual, 100)
				Convey("Remove", func() {
					So(db.RemovePresent(p.Id), ShouldBeNil)
					_, err := db.GetPresentByType(p.Title)
					So(err, ShouldNotBeNil)
				})
			})
			Convey("Update", func() {
				p.Title = "lel"
				present, err := db.UpdatePresent(p.Id, p)
				So(err, ShouldBeNil)
				So(present.Title, ShouldEqual, p.Title)
				Convey("Find by title", func() {
					p, err := db.GetPresentByType(p.Title)
					So(err, ShouldBeNil)
					So(present.Title, ShouldEqual, p.Title)
				})
			})
			Convey("Find all", func() {
				presents, err := db.GetAllPresents()
				So(err, ShouldBeNil)
				So(len(presents), ShouldEqual, 1)
				So(presents[0].Title, ShouldEqual, p.Title)
			})
			Convey("Send", func() {
				origin, destination := bson.NewObjectId(), bson.NewObjectId()
				So(db.Add(&models.User{Name: "Origin", Id: origin}), ShouldBeNil)
				So(db.Add(&models.User{Name: "Destination", Id: destination}), ShouldBeNil)
				event, err := db.SendPresent(origin, destination, p.Title)
				So(err, ShouldBeNil)
				So(event.Type, ShouldEqual, p.Title)
				Convey("Recieve", func() {
					presents, err := db.GetUserPresents(destination)
					So(err, ShouldBeNil)
					So(len(presents), ShouldEqual, 1)
					So(presents[0].Type, ShouldEqual, p.Title)
				})
			})
		})
	})
}
