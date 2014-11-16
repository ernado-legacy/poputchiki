package database

import (
	"testing"
	"time"

	. "github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
)

var (
	OfflineTimeout = time.Hour * 24
)

func TestDBMethods(t *testing.T) {
	var err error
	db := TestDatabase()
	u := User{}

	Convey("Database init", t, func() {
		Reset(func() {
			db.Drop()
		})
		Convey("User should be created", func() {
			u.Password = "test"
			u.Id = bson.NewObjectId()
			id := u.Id
			u.Birthday = time.Now().AddDate(-25, 0, 0)
			u.Email = "test@" + "test"
			u.Sex = SexMale
			u.Growth = 180
			u.Seasons = []string{SeasonSummer, SeasonSpring}
			u.Subscriptions = []string{SubscriptionLikesPhoto, SubscriptionNews}
			err = db.Add(&u)
			So(err, ShouldBeNil)

			Convey("Offline timeout", func() {
				timedOut := time.Now().Add(-OfflineTimeout - time.Second)
				So(db.SetOnline(id), ShouldNotBeNil)
				_, err := db.Update(id, bson.M{"lastaction": timedOut})
				So(err, ShouldBeNil)
				user := db.Get(id)
				So(user.Online, ShouldBeTrue)
				So(user.LastAction.Unix(), ShouldAlmostEqual, timedOut.Unix())
				_, err = db.UpdateAllStatuses()
				So(err, ShouldBeNil)
				user = db.Get(id)
				So(user.Online, ShouldBeFalse)
			})

			Convey("Subscriptions", func() {
				v, err := db.UserIsSubscribed(id, SubscriptionLikesPhoto)
				So(err, ShouldBeNil)
				So(v, ShouldBeTrue)
				v, err = db.UserIsSubscribed(id, SubscriptionNews)
				So(err, ShouldBeNil)
				So(v, ShouldBeTrue)
				v, err = db.UserIsSubscribed(id, SubscriptionLikesStatus)
				So(err, ShouldBeNil)
				So(v, ShouldBeFalse)
			})

			Convey("Confirmation", func() {
				token := db.NewConfirmationToken(id)
				So(token, ShouldNotBeNil)

				token1 := db.GetConfirmationToken(token.Token)
				So(token1, ShouldNotBeNil)

				token2 := db.GetConfirmationToken(token.Token)
				So(token2, ShouldBeNil)

				Convey("Phone", func() {
					So(db.ConfirmPhone(id), ShouldBeNil)
					user := db.Get(id)
					So(user, ShouldNotBeNil)
					So(user.PhoneConfirmed, ShouldBeTrue)
				})
				Convey("Email", func() {
					So(db.ConfirmEmail(id), ShouldBeNil)
					user := db.Get(id)
					So(user, ShouldNotBeNil)
					So(user.EmailConfirmed, ShouldBeTrue)
				})
			})

			Convey("Balance update", func() {
				So(db.IncBalance(id, 100), ShouldBeNil)
				So(db.DecBalance(id, 50), ShouldBeNil)
				newU := db.Get(id)
				So(newU.Balance, ShouldEqual, 50)
				So(db.DecBalance(id, 100), ShouldNotBeNil)
			})
			Convey("Search", func() {
				p := Pagination{}
				Convey("Growth", func() {
					result, _, err := db.Search(&SearchQuery{GrowthMin: 175, GrowthMax: 186}, p)
					So(err, ShouldBeNil)
					found := false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeTrue)
					result, _, err = db.Search(&SearchQuery{GrowthMin: 160, GrowthMax: 179}, p)
					So(err, ShouldBeNil)
					found = false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeFalse)
				})
				Convey("Season", func() {
					result, _, err := db.Search(&SearchQuery{Seasons: []string{SeasonAutumn, SeasonSummer}}, p)
					So(err, ShouldBeNil)
					found := false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeTrue)
					result, _, err = db.Search(&SearchQuery{Seasons: []string{SeasonWinter, SeasonAutumn}}, p)
					So(err, ShouldBeNil)
					found = false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeFalse)
				})
				Convey("Sex", func() {
					result, _, err := db.Search(&SearchQuery{Sex: SexMale}, p)
					So(err, ShouldBeNil)
					found := false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeTrue)
					result, _, err = db.Search(&SearchQuery{Sex: SexFemale}, p)
					So(err, ShouldBeNil)
					found = false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeFalse)
				})
				Convey("Age", func() {
					result, _, err := db.Search(&SearchQuery{AgeMin: 18, AgeMax: 26}, p)
					So(err, ShouldBeNil)
					found := false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeTrue)
					result, _, err = db.Search(&SearchQuery{AgeMax: 23}, p)
					So(err, ShouldBeNil)
					found = false
					for _, item := range result {
						if item.Id == u.Id {
							found = true
						}
					}
					So(found, ShouldBeFalse)
				})
			})
			Convey("Status update", func() {
				db.SetOnline(id)
				newU1 := db.Get(id)
				db.SetOffline(id)
				newU2 := db.Get(id)
				So(newU1.Online, ShouldEqual, true)
				So(newU2.Online, ShouldEqual, false)
			})

			Convey("Stripe add", func() {
				video := Video{}
				video.Id = bson.NewObjectId()
				video.User = id

				s := &StripeItem{}
				s.Id = bson.NewObjectId()
				s.User = id
				s, err := db.AddStripeItem(s, video)
				So(err, ShouldBeNil)
				Convey("Stripe get", func() {
					s1, err := db.GetStripeItem(s.Id)
					So(err, ShouldBeNil)
					So(s1.Type, ShouldEqual, "video")
					dta, err := bson.Marshal(s1.Media)
					So(err, ShouldBeNil)
					So(bson.Unmarshal(dta, &video), ShouldBeNil)
				})
				Convey("In stripe", func() {
					stripe, err := db.GetStripe(0, 0)
					So(err, ShouldBeNil)
					found := false
					for _, item := range stripe {
						if item.Id == s.Id {
							found = true
						}
					}
					So(found, ShouldBeTrue)
				})
			})

			Convey("Status add", func() {
				text := "status hello world"
				s, err := db.AddStatus(id, text)
				So(err, ShouldBeNil)
				So(s.Text, ShouldEqual, text)
				Convey("Remove", func() {
					err = db.RemoveStatusSecure(id, s.Id)
					So(err, ShouldBeNil)
				})
				Convey("Remove is secure", func() {
					err = db.RemoveStatusSecure(bson.NewObjectId(), s.Id)
					So(err, ShouldNotBeNil)
				})
				Convey("Update", func() {
					newText := "status2"
					s1, err := db.UpdateStatusSecure(id, s.Id, newText)
					So(err, ShouldBeNil)
					s2, err := db.GetStatus(s.Id)
					So(err, ShouldBeNil)
					So(s1.Text, ShouldEqual, newText)
					So(s2.Text, ShouldEqual, newText)
					So(s1.Id, ShouldEqual, s.Id)
					So(s2.Id, ShouldEqual, s.Id)
					So(s1.Text, ShouldNotEqual, s.Text)
				})
				Convey("Exists", func() {
					s1, err := db.GetCurrentStatus(id)
					So(err, ShouldBeNil)
					So(s1.Text, ShouldEqual, text)
				})
				Convey("Searchable", func() {
					query := &SearchQuery{}
					query.Sex = "male"
					statuses, err := db.SearchStatuses(query, Pagination{})
					So(err, ShouldBeNil)
					So(len(statuses), ShouldEqual, 1)
					So(statuses[0].Id, ShouldEqual, s.Id)
				})
				Convey("Actual", func() {
					newText := "status actual"
					_, err := db.AddStatus(id, newText)
					So(err, ShouldBeNil)
					time.Sleep(200 * time.Millisecond)
					s2, err := db.GetCurrentStatus(id)
					So(err, ShouldBeNil)
					So(s2.Text, ShouldEqual, newText)
					statuses, err := db.GetLastStatuses(10)
					So(err, ShouldBeNil)
					So(statuses[0].Text, ShouldEqual, newText)
				})

			})
			Convey("Add photo", func() {
				p, err := db.AddPhoto(id, "", "")
				So(err, ShouldBeNil)
				Convey("Remove", func() {
					err := db.RemovePhoto(id, p.Id)
					So(err, ShouldBeNil)
				})

				Convey("Like", func() {
					So(db.AddLikePhoto(id, p.Id), ShouldBeNil)
					photo, err := db.GetPhoto(p.Id)
					So(err, ShouldBeNil)
					So(photo.Likes, ShouldEqual, 1)

					likers := db.GetLikesPhoto(photo.Id)
					found := false

					for _, liker := range likers {
						if liker.Id == id {
							found = true
						}
					}

					So(found, ShouldBeTrue)

					Convey("Unlike", func() {
						So(db.RemoveLikePhoto(id, p.Id), ShouldBeNil)
						photo, err := db.GetPhoto(p.Id)
						So(err, ShouldBeNil)
						So(photo.Likes, ShouldEqual, 0)
						likers := db.GetLikesPhoto(photo.Id)
						found := false

						for _, liker := range likers {
							if liker.Id == id {
								found = true
							}
						}

						So(found, ShouldBeFalse)
					})
				})

				Convey("Search", func() {
					query := &SearchQuery{}
					query.Sex = "male"
					photos, err := db.SearchPhoto(query, Pagination{})
					So(err, ShouldBeNil)

					found := false
					for k := range photos {
						if photos[k].Id == p.Id {
							found = true
						}
					}
					So(found, ShouldBeTrue)
				})
			})
			Convey("Add video", func() {
				video := &Video{}
				video.Id = bson.NewObjectId()
				video.User = id
				v, err := db.AddVideo(video)
				So(err, ShouldBeNil)

				Convey("Like", func() {
					So(db.AddLikeVideo(id, v.Id), ShouldBeNil)
					video := db.GetVideo(v.Id)
					So(video.Likes, ShouldEqual, 1)

					likers := db.GetLikesVideo(video.Id)
					found := false

					for _, liker := range likers {
						if liker.Id == id {
							found = true
						}
					}

					So(found, ShouldBeTrue)

					Convey("Unlike", func() {
						So(db.RemoveLikeVideo(id, v.Id), ShouldBeNil)
						video := db.GetVideo(v.Id)
						So(err, ShouldBeNil)
						So(video.Likes, ShouldEqual, 0)
						likers := db.GetLikesVideo(video.Id)
						found := false

						for _, liker := range likers {
							if liker.Id == id {
								found = true
							}
						}

						So(found, ShouldBeFalse)
					})
				})
			})
		})
	})
}
