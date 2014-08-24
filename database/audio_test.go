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
		u := &models.User{Id: bson.NewObjectId(), Name: "Alex", Email: "test@kek.ru", Sex: models.SexMale}
		So(db.Add(u), ShouldBeNil)
		Convey("Integrity", Integrity(db, u))
		Convey("Add", func() {
			a := &models.Audio{}
			a.Id = bson.NewObjectId()
			a.User = u.Id
			_, err := db.AddAudio(a)
			So(err, ShouldBeNil)
			Convey("Get", func() {
				b := db.GetAudio(a.Id)
				So(b.User, ShouldEqual, a.User)
			})
			Convey("User get", func() {
				newUser := db.Get(u.Id)
				So(newUser.Audio, ShouldEqual, a.Id)
			})
			Convey("Integrity", Integrity(db, u))
			Convey("Update", func() {
				fidAac := "1231424123"
				fidOgg := "j980234089dsf"
				So(db.UpdateAudioAAC(a.Id, fidAac), ShouldBeNil)
				So(db.UpdateAudioOGG(a.Id, fidOgg), ShouldBeNil)
				b := db.GetAudio(a.Id)
				So(b.User, ShouldEqual, a.User)
				So(b.AudioAac, ShouldEqual, fidAac)
				So(b.AudioOgg, ShouldEqual, fidOgg)
				Convey("Integrity", Integrity(db, u))
				Convey("User get", func() {
					newUser := db.Get(u.Id)
					So(newUser.Audio, ShouldEqual, a.Id)
					So(newUser.AudioAAC, ShouldEqual, fidAac)
					So(newUser.AudioOGG, ShouldEqual, fidOgg)
				})
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
