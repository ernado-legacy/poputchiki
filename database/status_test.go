package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
	"time"
)

func Integrity(db models.DataBase, u *models.User) func() {
	return func() {
		dbu := db.Get(u.Id)
		So(dbu, ShouldNotBeNil)
		So(dbu.Email, ShouldEqual, u.Email)
		So(dbu.Name, ShouldEqual, u.Name)
		So(dbu.Sex, ShouldEqual, u.Sex)
		So(dbu.Phone, ShouldEqual, u.Phone)
		So(dbu.Rating, ShouldAlmostEqual, u.Rating)
		So(dbu.Vip, ShouldEqual, u.Vip)
		So(dbu.LastAction.Unix(), ShouldEqual, u.LastAction.Unix())
	}
}

func TestStatus(t *testing.T) {
	db := TestDatabase()
	Convey("Status test", t, func() {
		Reset(db.Drop)
		id := bson.NewObjectId()
		email := "test"
		u := &models.User{Id: id, Email: email}
		Convey("Add user", func() {
			So(db.Add(u), ShouldBeNil)
			text := "кекс в рот"
			Convey("Add status", func() {
				s, err := db.AddStatus(u.Id, text)
				So(err, ShouldBeNil)
				So(s.Text, ShouldEqual, text)
				Convey("Integrity", Integrity(db, u))
				Convey("Last day statuses", func() {
					count, err := db.GetLastDayStatusesAmount(id)
					So(err, ShouldBeNil)
					So(count, ShouldEqual, 1)
				})
				Convey("Likes", func() {
					newUser := &models.User{Id: bson.NewObjectId()}
					So(db.Add(newUser), ShouldBeNil)
					Convey("Add", func() {
						So(db.AddLikeStatus(newUser.Id, s.Id), ShouldBeNil)
						status, err := db.GetStatus(s.Id)
						So(err, ShouldBeNil)
						So(status.Likes, ShouldEqual, 1)
						So(status.LikedUsers[0], ShouldEqual, newUser.Id)
						Convey("Integrity", Integrity(db, u))
						Convey("Search", func() {
							statuses, err := db.SearchStatuses(new(models.SearchQuery), 0, 0)
							So(err, ShouldBeNil)
							So(len(statuses), ShouldEqual, 1)
							So(statuses[0].Likes, ShouldEqual, 1)
						})
						Convey("Top", func() {
							statuses, err := db.GetTopStatuses(1, 0)
							So(err, ShouldBeNil)
							So(len(statuses), ShouldEqual, 1)
							So(statuses[0].Likes, ShouldEqual, 1)
							So(statuses[0].Id, ShouldEqual, s.Id)
						})
						Convey("Current status", func() {
							status, err := db.GetCurrentStatus(id)
							So(err, ShouldBeNil)
							So(status.Likes, ShouldEqual, 1)
						})
						Convey("Likers", func() {
							likers := db.GetLikesStatus(s.Id)
							So(likers, ShouldNotBeNil)
							So(len(likers), ShouldEqual, 1)
							So(likers[0].Id, ShouldEqual, newUser.Id)
						})
						Convey("Remove", func() {
							So(db.RemoveLikeStatus(newUser.Id, s.Id), ShouldBeNil)
							status, err := db.GetStatus(s.Id)
							So(err, ShouldBeNil)
							So(status.Likes, ShouldEqual, 0)
							So(len(status.LikedUsers), ShouldEqual, 0)
							Convey("Integrity", Integrity(db, u))
							Convey("Search", func() {
								statuses, err := db.SearchStatuses(new(models.SearchQuery), 1, 0)
								So(err, ShouldBeNil)
								So(len(statuses), ShouldEqual, 1)
								So(statuses[0].Likes, ShouldEqual, 0)
							})
							Convey("Current status", func() {
								status, err := db.GetCurrentStatus(id)
								So(err, ShouldBeNil)
								So(status.Likes, ShouldEqual, 0)
							})
							Convey("Likers", func() {
								likers := db.GetLikesStatus(s.Id)
								So(likers, ShouldBeNil)
							})
						})
					})
				})
				Convey("Status ok", func() {
					s, err := db.GetStatus(s.Id)
					So(err, ShouldBeNil)
					So(s.Text, ShouldEqual, text)
				})
				Convey("Current status", func() {
					s, err := db.GetCurrentStatus(id)
					So(err, ShouldBeNil)
					So(s.Text, ShouldEqual, text)
				})
				Convey("User status updated", func() {
					user := db.Get(id)
					So(user, ShouldNotBeNil)
					So(user.Status, ShouldEqual, text)
					So(user.StatusUpdate.Truncate(time.Second).Unix(), ShouldEqual, s.Time.Truncate(time.Second).Unix())
				})
				Convey("Search", func() {
					Convey("All", func() {
						statuses, err := db.SearchStatuses(new(models.SearchQuery), 1, 0)
						So(err, ShouldBeNil)
						So(len(statuses), ShouldEqual, 1)
						So(statuses[0].Text, ShouldEqual, text)

					})
				})
				Convey("Change status", func() {
					newtext := "kekekke"
					s, err := db.UpdateStatusSecure(id, s.Id, newtext)
					So(err, ShouldBeNil)
					So(s.Text, ShouldEqual, newtext)
					s, err = db.GetStatus(s.Id)
					So(err, ShouldBeNil)
					So(s.Text, ShouldEqual, newtext)
					Convey("Integrity", Integrity(db, u))
					Convey("User status updated", func() {
						user := db.Get(id)
						So(user, ShouldNotBeNil)
						So(user.Status, ShouldEqual, newtext)
						So(user.StatusUpdate.Truncate(time.Second).Unix(), ShouldEqual, s.Time.Truncate(time.Second).Unix())
					})
					Convey("Current status", func() {
						s, err := db.GetCurrentStatus(id)
						So(err, ShouldBeNil)
						So(s.Text, ShouldEqual, newtext)
					})
					Convey("Search", func() {
						statuses, err := db.SearchStatuses(new(models.SearchQuery), 1, 0)
						So(err, ShouldBeNil)
						So(len(statuses), ShouldEqual, 1)
						So(statuses[0].Text, ShouldEqual, newtext)
					})
				})
				Convey("Add another status", func() {
					text := "asdasdas123123"
					s, err := db.AddStatus(u.Id, text)
					So(err, ShouldBeNil)
					So(s.Text, ShouldEqual, text)
					Convey("Integrity", Integrity(db, u))
					Convey("Last day statuses", func() {
						count, err := db.GetLastDayStatusesAmount(id)
						So(err, ShouldBeNil)
						So(count, ShouldEqual, 2)
					})
					Convey("Status ok", func() {
						s, err := db.GetStatus(s.Id)
						So(err, ShouldBeNil)
						So(s.Text, ShouldEqual, text)
					})
					Convey("Current status", func() {
						s, err := db.GetCurrentStatus(id)
						So(err, ShouldBeNil)
						So(s.Text, ShouldEqual, text)
					})
					Convey("User status updated", func() {
						user := db.Get(id)
						So(user, ShouldNotBeNil)
						So(user.Status, ShouldEqual, text)
						So(user.StatusUpdate.Truncate(time.Second).Unix(), ShouldEqual, s.Time.Truncate(time.Second).Unix())
					})
					Convey("Search", func() {
						statuses, err := db.SearchStatuses(new(models.SearchQuery), 1, 0)
						So(err, ShouldBeNil)
						So(len(statuses), ShouldEqual, 2)
						So(statuses[0].Text, ShouldEqual, text)
					})
					Convey("Change status", func() {
						newtext := "cvzxvxcvzxcv"
						s, err := db.UpdateStatusSecure(id, s.Id, newtext)
						So(err, ShouldBeNil)
						So(s.Text, ShouldEqual, newtext)
						s, err = db.GetStatus(s.Id)
						So(err, ShouldBeNil)
						So(s.Text, ShouldEqual, newtext)
						Convey("Integrity", Integrity(db, u))
						Convey("User status updated", func() {
							user := db.Get(id)
							So(user, ShouldNotBeNil)
							So(user.Status, ShouldEqual, newtext)
							So(user.StatusUpdate.Truncate(time.Second).Unix(), ShouldEqual, s.Time.Truncate(time.Second).Unix())
						})
						Convey("Current status", func() {
							s, err := db.GetCurrentStatus(id)
							So(err, ShouldBeNil)
							So(s.Text, ShouldEqual, newtext)
						})
						Convey("Search", func() {
							statuses, err := db.SearchStatuses(new(models.SearchQuery), 1, 0)
							So(err, ShouldBeNil)
							So(len(statuses), ShouldEqual, 2)
							So(statuses[0].Text, ShouldEqual, newtext)
						})
					})
				})
			})
		})
	})

}
