package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestAudio(t *testing.T) {
	db := TestDatabase()
	Convey("Audio", t, func() {
		Reset(db.Drop)
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
			Convey("Update", func() {
				fidAac := "1231424123"
				fidOgg := "j980234089dsf"
				So(db.UpdateAudioAAC(a.Id, fidAac), ShouldBeNil)
				So(db.UpdateAudioOGG(a.Id, fidOgg), ShouldBeNil)
				b := db.GetAudio(a.Id)
				So(b.User, ShouldEqual, a.User)
				So(b.AudioAac, ShouldEqual, fidAac)
				So(b.AudioOgg, ShouldEqual, fidOgg)
			})
		})
		Convey("Update not found", func() {
			fidAac := "1231424123"
			fidOgg := "j980234089dsf"
			So(db.UpdateAudioAAC(bson.NewObjectId(), fidAac), ShouldEqual, mgo.ErrNotFound)
			So(db.UpdateAudioOGG(bson.NewObjectId(), fidOgg), ShouldEqual, mgo.ErrNotFound)
		})
		Convey("Get nill", func() {
			So(db.GetAudio(bson.NewObjectId()), ShouldBeNil)
		})
	})

}
