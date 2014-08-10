package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestVideo(t *testing.T) {
	db := TestDatabase()
	Convey("Audio", t, func() {
		Reset(func() {
			db.Drop()
		})
		Convey("Add", func() {
			a := &models.Video{}
			a.Id = bson.NewObjectId()
			a.User = bson.NewObjectId()
			_, err := db.AddVideo(a)
			So(err, ShouldBeNil)
			Convey("Get", func() {
				b := db.GetVideo(a.Id)
				So(b.User, ShouldEqual, a.User)
			})
			Convey("Add like", func() {
				So(db.AddLikeVideo(a.User, a.Id), ShouldBeNil)
				Convey("Liked", func() {
					c := db.GetVideo(a.Id)
					So(c.Likes, ShouldEqual, 1)
				})
			})
		})
	})

}
