package database

import (
	"testing"

	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func TestAdvertisments(t *testing.T) {
	db := TestDatabase()
	Convey("Advertisments", t, func() {
		Reset(db.Drop)
		Convey("Add photo", func() {
			text := "test text 222"
			p := &models.Photo{}
			p.Id = bson.NewObjectId()
			i := &models.StripeItem{}
			i.Text = text
			p.User = bson.NewObjectId()
			a, err := db.AddAdvertisement(bson.NewObjectId(), i, p)
			So(err, ShouldBeNil)
			So(a.Type, ShouldEqual, "photo")
			Convey("Find", func() {
				ads, err := db.GetAds(0, 0)
				So(err, ShouldBeNil)
				So(len(ads), ShouldEqual, 1)
			})
			Convey("Get", func() {
				ad, err := db.GetAdvertisment(a.Id)
				So(err, ShouldBeNil)
				So(ad.Text, ShouldEqual, text)
			})
		})
		Convey("Add", func() {
			text := "test text"
			i := &models.StripeItem{}
			i.Text = text
			a, err := db.AddAdvertisement(bson.NewObjectId(), i, nil)
			So(err, ShouldBeNil)
			So(a.Type, ShouldEqual, "text")
			Convey("Find", func() {
				ads, err := db.GetAds(0, 0)
				So(err, ShouldBeNil)
				So(len(ads), ShouldEqual, 1)
			})
			Convey("Get", func() {
				ad, err := db.GetAdvertisment(a.Id)
				So(err, ShouldBeNil)
				So(ad.Text, ShouldEqual, text)
			})
			Convey("Remove", func() {
				So(db.RemoveAdvertisment(a.User, a.Id), ShouldBeNil)
				_, err := db.GetAdvertisment(a.Id)
				So(err, ShouldEqual, mgo.ErrNotFound)
			})
		})
	})
}
