package database

import (
	"log"
	"testing"

	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
)

func TestVideo(t *testing.T) {
	db := TestDatabase()
	Convey("Video", t, func() {
		Reset(db.Drop)
		Convey("Get nil", func() {
			So(db.GetVideo(bson.NewObjectId()), ShouldBeNil)
		})
		Convey("Add", func() {
			a := &models.Video{}
			a.Id = bson.NewObjectId()
			u := &models.User{Id: bson.NewObjectId()}
			a.User = u.Id
			_, err := db.AddVideo(a)
			So(db.Add(u), ShouldBeNil)
			So(err, ShouldBeNil)
			Convey("Get", func() {
				b := db.GetVideo(a.Id)
				So(b.User, ShouldEqual, a.User)
			})
			Convey("Get users video", func() {
				v, err := db.GetUserVideo(a.User)
				So(err, ShouldBeNil)
				So(len(v), ShouldEqual, 1)
				So(v[0].Id, ShouldEqual, a.Id)
			})
			Convey("Add like", func() {
				So(db.AddLikeVideo(a.User, a.Id), ShouldBeNil)
				Convey("Liked", func() {
					c := db.GetVideo(a.Id)
					So(c.Likes, ShouldEqual, 1)
					log.Println(c.LikedUsers)
					So(c.LikedUsers, ShouldContain, a.User)
				})
				Convey("Likers", func() {
					likers := db.GetLikesVideo(a.Id)
					found := false
					for _, user := range likers {
						if user.Id == a.User {
							found = true
						}
					}
					So(found, ShouldBeTrue)
				})
			})
			Convey("Remove", func() {
				So(db.RemoveVideo(a.User, a.Id), ShouldBeNil)
				So(db.GetVideo(a.Id), ShouldBeNil)
			})
			Convey("Update", func() {
				fidMpeg := "1231424123"
				fidWebm := "j980234089dsf"
				fidJpeg := "fdgdfg123213sf"
				So(db.UpdateVideoMpeg(a.Id, fidMpeg), ShouldBeNil)
				So(db.UpdateVideoWebm(a.Id, fidWebm), ShouldBeNil)
				So(db.UpdateVideoThumbnail(a.Id, fidJpeg), ShouldBeNil)
				b := db.GetVideo(a.Id)
				So(b.User, ShouldEqual, a.User)
				So(b.VideoMpeg, ShouldEqual, fidMpeg)
				So(b.VideoWebm, ShouldEqual, fidWebm)
				So(b.ThumbnailJpeg, ShouldEqual, fidJpeg)
			})
		})
	})

}
