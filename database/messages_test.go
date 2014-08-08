package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"labix.org/v2/mgo/bson"
	"testing"
	"time"
)

func TestMessages(t *testing.T) {
	db := TestDatabase()
	origin := bson.NewObjectId()
	destination := bson.NewObjectId()
	text := "Привет"

	Convey("Add users", t, func() {
		Reset(func() {
			db.Drop()
		})
		uOrigin := &models.User{Id: origin}
		uDestination := &models.User{Id: destination}
		So(db.Add(uOrigin), ShouldBeNil)
		So(db.Add(uDestination), ShouldBeNil)
		Convey("Add message", func() {
			now := time.Now()
			idOrigin := bson.NewObjectId()
			idDestination := bson.NewObjectId()
			mOrigin := &models.Message{idOrigin, destination, origin, origin, destination, false, now, text, false}
			mDestination := &models.Message{idDestination, origin, destination, origin, destination, false, now, text, false}
			So(db.AddMessage(mOrigin), ShouldBeNil)
			So(db.AddMessage(mDestination), ShouldBeNil)
			Convey("Destination chats", func() {
				chats, err := db.GetChats(destination)
				So(err, ShouldBeNil)
				found := false
				for k := range chats {
					if chats[k].Id == origin {
						found = true
					}
				}
				So(found, ShouldBeTrue)
			})
			Convey("Add new message", func() {
				text2 := "hehehe"
				now := time.Now()
				idOrigin := bson.NewObjectId()
				idDestination := bson.NewObjectId()
				mOrigin := &models.Message{idOrigin, destination, origin, origin, destination, false, now, text2, false}
				mDestination := &models.Message{idDestination, origin, destination, origin, destination, false, now, text2, false}
				So(db.AddMessage(mOrigin), ShouldBeNil)
				So(db.AddMessage(mDestination), ShouldBeNil)
				Convey("Destination chats", func() {
					chats, err := db.GetChats(destination)
					So(err, ShouldBeNil)
					found := false
					for k := range chats {
						if chats[k].Id == origin {
							found = true
						}
					}
					So(found, ShouldBeTrue)
					Convey("Destination has new message", func() {
						n, err := db.GetUnreadCount(destination)
						So(n, ShouldEqual, 2)
						So(err, ShouldBeNil)
						Convey("Fist message should be second", func() {
							messages, err := db.GetMessagesFromUser(destination, origin)
							So(err, ShouldBeNil)
							So(len(messages), ShouldEqual, 2)
							So(messages[1].Text, ShouldEqual, text2)
						})
					})
					Convey("Origin has new message", func() {
						n, err := db.GetUnreadCount(origin)
						So(n, ShouldEqual, 2)
						So(err, ShouldBeNil)
					})

				})

				Convey("Add new message", func() {
					text3 := "hehehe"
					now := time.Now()
					idOrigin := bson.NewObjectId()
					idDestination := bson.NewObjectId()
					mOrigin := &models.Message{idOrigin, destination, origin, origin, destination, false, now, text3, false}
					mDestination := &models.Message{idDestination, origin, destination, origin, destination, false, now, text3, false}
					So(db.AddMessage(mOrigin), ShouldBeNil)
					So(db.AddMessage(mDestination), ShouldBeNil)
					Convey("Destination chats", func() {
						chats, err := db.GetChats(destination)
						So(err, ShouldBeNil)
						found := false
						for k := range chats {
							if chats[k].Id == origin {
								found = true
							}
						}
						So(found, ShouldBeTrue)
						Convey("Destination has new message", func() {
							n, err := db.GetUnreadCount(destination)
							So(n, ShouldEqual, 3)
							So(err, ShouldBeNil)
							Convey("Fist message should be second", func() {
								messages, err := db.GetMessagesFromUser(destination, origin)
								So(err, ShouldBeNil)
								So(len(messages), ShouldEqual, 3)
								So(messages[2].Text, ShouldEqual, text3)
							})
						})
						Convey("Origin has new message", func() {
							n, err := db.GetUnreadCount(origin)
							So(n, ShouldEqual, 3)
							So(err, ShouldBeNil)
						})

					})
				})
			})
			Convey("Origin chats", func() {
				chats, err := db.GetChats(origin)
				So(err, ShouldBeNil)
				found := false
				for k := range chats {
					if chats[k].Id == destination {
						found = true
					}
				}
				So(found, ShouldBeTrue)
			})
			Convey("Destination has new message", func() {
				n, err := db.GetUnreadCount(destination)
				So(n, ShouldEqual, 1)
				So(err, ShouldBeNil)
			})
			Convey("Origin has new message", func() {
				n, err := db.GetUnreadCount(origin)
				So(n, ShouldEqual, 1)
				So(err, ShouldBeNil)
			})
			Convey("Read origin", func() {
				So(db.SetRead(origin, idOrigin), ShouldBeNil)
				m, err := db.GetMessage(idOrigin)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeTrue)
				Convey("Origin has no new messages", func() {
					n, err := db.GetUnreadCount(origin)
					So(n, ShouldEqual, 0)
					So(err, ShouldBeNil)
				})
			})
			Convey("Read destination", func() {
				So(db.SetRead(destination, idDestination), ShouldBeNil)
				m, err := db.GetMessage(idDestination)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeTrue)
				Convey("Destination has no new messages", func() {
					n, err := db.GetUnreadCount(destination)
					So(n, ShouldEqual, 0)
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
