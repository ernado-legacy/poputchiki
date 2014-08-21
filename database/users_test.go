package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
	"time"
)

func TestUsers(t *testing.T) {
	db := TestDatabase()
	Convey("Users", t, func() {
		Reset(db.Drop)
		id := bson.NewObjectId()
		u := &models.User{Id: id, Email: "Keks", Name: "Lalka", Sex: models.SexMale, Rating: 100.0}
		Convey("Null get", func() {
			So(db.Get(bson.NewObjectId()), ShouldBeNil)
			So(db.GetUsername("asdasdasasd"), ShouldBeNil)
		})
		Convey("Add", func() {
			So(db.Add(u), ShouldBeNil)
			Convey("Integrity", Integrity(db, u))
			Convey("Get by username", func() {
				So(db.GetUsername(u.Email), ShouldNotBeNil)
			})
			Convey("Add guest", func() {
				guestId := bson.NewObjectId()
				guest := &models.User{Id: guestId}
				So(db.Add(guest), ShouldBeNil)
				So(db.AddGuest(id, guestId), ShouldBeNil)
				Convey("Integrity", Integrity(db, u))
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
			Convey("Search", func() {
				q := new(models.SearchQuery)
				q.Sex = models.SexMale
				users, count, err := db.Search(q, models.Pagination{})
				So(err, ShouldBeNil)
				So(len(users), ShouldEqual, 1)
				So(count, ShouldEqual, 1)
				So(users[0].Name, ShouldEqual, u.Name)
			})
			Convey("Update", func() {
				Convey("Name", func() {
					u.Name = "Alex"
					_, err := db.Update(id, bson.M{"name": u.Name})
					So(err, ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
					Convey("Search", func() {
						q := new(models.SearchQuery)
						q.Sex = models.SexMale
						users, count, err := db.Search(q, models.Pagination{})
						So(err, ShouldBeNil)
						So(len(users), ShouldEqual, 1)
						So(count, ShouldEqual, 1)
						So(users[0].Name, ShouldEqual, u.Name)
					})
				})
				Convey("Sex", func() {
					u.Sex = models.SexFemale
					_, err := db.Update(id, bson.M{"sex": u.Sex})
					So(err, ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
					Convey("Search", func() {
						q := new(models.SearchQuery)
						q.Sex = models.SexFemale
						users, count, err := db.Search(q, models.Pagination{})
						So(err, ShouldBeNil)
						So(len(users), ShouldEqual, 1)
						So(count, ShouldEqual, 1)
						So(users[0].Name, ShouldEqual, u.Name)
					})
				})
			})
			Convey("VIP", func() {
				Convey("Default disabled", func() {
					user := db.Get(id)
					So(user.Vip, ShouldBeFalse)
				})
				Convey("Enable", func() {
					u.Vip = true
					So(db.SetVip(id, true), ShouldBeNil)
					So(db.SetVipTill(id, time.Now().Add(-time.Second)), ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
					user := db.Get(id)
					So(user.Vip, ShouldBeTrue)
				})
			})
			Convey("Favourites", func() {
				Convey("Default blank", func() {
					user := db.Get(id)
					So(len(user.Favorites), ShouldEqual, 0)
				})
				Convey("Add", func() {
					favourite := &models.User{Id: bson.NewObjectId(), Email: "erasd@asdasd.sad", Name: "Kekes", Sex: models.SexFemale}
					So(db.Add(favourite), ShouldBeNil)
					So(db.AddToFavorites(id, favourite.Id), ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
					Convey("Should be in list", func() {
						user := db.Get(id)
						So(len(user.Favorites), ShouldEqual, 1)
						So(user.Favorites[0], ShouldEqual, favourite.Id)
					})
					Convey("Should be in full list", func() {
						favourites := db.GetFavorites(id)
						So(len(favourites), ShouldEqual, 1)
						So(favourites[0].Id, ShouldEqual, favourite.Id)
					})
					Convey("Should be in followers", func() {
						followers, err := db.GetAllUsersWithFavorite(favourite.Id)
						So(err, ShouldBeNil)
						So(len(followers), ShouldEqual, 1)
						So(followers[0].Id, ShouldEqual, id)
					})
					Convey("Remove", func() {
						So(db.RemoveFromFavorites(id, favourite.Id), ShouldBeNil)
						Convey("Integrity", Integrity(db, u))
						Convey("Should not be in list", func() {
							user := db.Get(id)
							So(len(user.Favorites), ShouldEqual, 0)
						})
						Convey("Should not be in full list", func() {
							favourites := db.GetFavorites(id)
							So(len(favourites), ShouldEqual, 0)
						})
						Convey("Should not be in followers", func() {
							followers, err := db.GetAllUsersWithFavorite(favourite.Id)
							So(err, ShouldBeNil)
							So(len(followers), ShouldEqual, 0)
						})
					})
				})
			})
			Convey("Blacklist", func() {
				Convey("Default blank", func() {
					user := db.Get(id)
					So(len(user.Blacklist), ShouldEqual, 0)
				})
				Convey("Add", func() {
					blacklisted := &models.User{Id: bson.NewObjectId(), Email: "erasd@asdasd.sad", Name: "Kekes", Sex: models.SexFemale}
					So(db.Add(blacklisted), ShouldBeNil)
					So(db.AddToBlacklist(id, blacklisted.Id), ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
					Convey("Should be in list", func() {
						user := db.Get(id)
						So(len(user.Blacklist), ShouldEqual, 1)
						So(user.Blacklist[0], ShouldEqual, blacklisted.Id)
					})
					Convey("Should be in full list", func() {
						blacklist := db.GetBlacklisted(id)
						So(len(blacklist), ShouldEqual, 1)
						So(blacklist[0].Id, ShouldEqual, blacklisted.Id)
					})
					Convey("Remove", func() {
						So(db.RemoveFromBlacklist(id, blacklisted.Id), ShouldBeNil)
						Convey("Integrity", Integrity(db, u))
						Convey("Should be in list", func() {
							user := db.Get(id)
							So(len(user.Blacklist), ShouldEqual, 0)
						})
						Convey("Should not be in full list", func() {
							blacklist := db.GetBlacklisted(id)
							So(len(blacklist), ShouldEqual, 0)
						})
					})
				})
			})
			Convey("Rating", func() {
				Convey("Set", func() {
					u.Rating = 50.0
					So(db.SetRating(id, 50.0), ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
				})
				Convey("Add", func() {
					u.Rating = 60
					So(db.SetRating(id, 50.0), ShouldBeNil)
					So(db.ChangeRating(id, 10), ShouldBeNil)
					Convey("Integrity", Integrity(db, u))
				})
				Convey("Degradation", func() {
					rate := 5.00
					u.Rating -= rate
					info, err := db.DegradeRating(rate)
					So(err, ShouldBeNil)
					So(info, ShouldNotBeNil)
					So(info.Updated, ShouldEqual, 1)
					Convey("Integrity", Integrity(db, u))
				})
				Convey("Normalization", func() {
					Convey("Above maximum", func() {
						u.Rating = 100.0
						So(db.SetRating(id, 200.0), ShouldBeNil)
						info, err := db.NormalizeRating()
						So(err, ShouldBeNil)
						So(info, ShouldNotBeNil)
						So(info.Updated, ShouldEqual, 1)
						Convey("Integrity", Integrity(db, u))
					})
					Convey("Below maximum", func() {
						u.Rating = 0.0
						So(db.SetRating(id, -200.0), ShouldBeNil)
						info, err := db.NormalizeRating()
						So(err, ShouldBeNil)
						So(info, ShouldNotBeNil)
						So(info.Updated, ShouldEqual, 1)
						Convey("Integrity", Integrity(db, u))
					})
				})
			})
			Convey("Last action", func() {
				u.LastAction = time.Now()
				So(db.SetLastActionNow(id), ShouldBeNil)
				Convey("Integrity", Integrity(db, u))
			})
			Convey("Set avatar", func() {
				u.Avatar = bson.NewObjectId()
				So(db.SetAvatar(id, u.Avatar), ShouldBeNil)
				Convey("Integrity", Integrity(db, u))
			})
		})
	})

}
