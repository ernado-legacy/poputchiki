package database

import (
	"log"
	"testing"

	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
)

func TestMessages(t *testing.T) {
	db := TestDatabase()
	origin := bson.NewObjectId()
	destination := bson.NewObjectId()
	text := "Привет"
	photo := "id12321.jpg"
	pagination := models.Pagination{}
	Convey("Add users", t, func() {
		Reset(db.Drop)
		uOrigin := &models.User{Id: origin, Name: "Alex"}
		uDestination := &models.User{Id: destination, Name: "Petya"}
		So(db.Add(uOrigin), ShouldBeNil)
		So(db.Add(uDestination), ShouldBeNil)
		Integrity(db, uOrigin)
		Integrity(db, uDestination)
		Convey("Add message", func() {
			mOrigin, mDestination := models.NewMessagePair(db, origin, destination, photo, text)
			idOrigin := mOrigin.Id
			idDestination := mDestination.Id
			So(db.AddMessage(mOrigin), ShouldBeNil)
			So(db.AddMessage(mDestination), ShouldBeNil)
			Integrity(db, uOrigin)
			Integrity(db, uDestination)
			Convey("Add notification", func() {
				_, err := db.AddUpdate(destination, origin, "messages", mDestination)
				So(err, ShouldBeNil)
				counters, err := db.GetUpdatesCount(destination)
				So(err, ShouldBeNil)
				So(len(counters), ShouldNotEqual, 1)
				found := false
				for _, v := range counters {
					if v.Type == "messages" {
						found = true
						So(v.Count, ShouldEqual, 1)
					}
				}
				So(found, ShouldBeTrue)
				Convey("Set read", func() {
					So(db.SetReadMessagesFromUser(destination, origin), ShouldBeNil)
					found := false
					counters, err := db.GetUpdatesCount(destination)
					So(err, ShouldBeNil)
					for _, v := range counters {
						if v.Type == "messages" {
							found = true
						}
					}
					updates, err := db.GetUpdates(destination, "messages", models.Pagination{})
					So(err, ShouldBeNil)
					for _, update := range updates {
						So(update.Read, ShouldBeTrue)
					}
					So(found, ShouldBeFalse)
				})
			})
			Convey("Destination chats", func() {
				chats, err := db.GetChats(destination)
				So(err, ShouldBeNil)
				found := false
				for k := range chats {
					if chats[k].Id == origin {
						So(chats[k].Unread, ShouldEqual, 1)
						found = true
					}
				}
				So(found, ShouldBeTrue)
			})
			Convey("Remove chats", func() {
				So(db.RemoveChat(destination, origin), ShouldBeNil)
				chats, err := db.GetChats(destination)
				So(err, ShouldBeNil)
				found := false
				for k := range chats {
					if chats[k].Id == origin {
						So(chats[k].Unread, ShouldEqual, 1)
						found = true
					}
				}
				So(found, ShouldBeFalse)
			})
			Convey("Last message id", func() {
				id, err := db.GetLastMessageIdFromUser(destination, origin)
				So(err, ShouldBeNil)
				log.Println(id, idDestination)
				So(id, ShouldEqual, idDestination)
			})
			Convey("Add new message", func() {
				text2 := "hehehe"
				mOrigin, mDestination := models.NewMessagePair(db, origin, destination, photo, text2)
				So(db.AddMessage(mOrigin), ShouldBeNil)
				So(db.AddMessage(mDestination), ShouldBeNil)
				Integrity(db, uOrigin)
				Integrity(db, uDestination)
				Convey("Set read origin", func() {
					So(db.SetReadMessagesFromUser(destination, origin), ShouldBeNil)
					Integrity(db, uOrigin)
					Integrity(db, uDestination)
					Convey("Destination has no new messages", func() {
						n, err := db.GetUnreadCount(destination)
						So(n, ShouldEqual, 0)
						So(err, ShouldBeNil)
					})
				})
				Convey("Remove", func() {
					So(db.RemoveMessage(idOrigin), ShouldBeNil)
					Integrity(db, uOrigin)
					Integrity(db, uDestination)
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
								messages, err := db.GetMessagesFromUser(destination, origin, pagination)
								So(err, ShouldBeNil)
								So(len(messages), ShouldEqual, 2)
								So(messages[1].Text, ShouldEqual, text2)
							})
						})
					})
				})
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
							messages, err := db.GetMessagesFromUser(destination, origin, pagination)
							So(err, ShouldBeNil)
							So(len(messages), ShouldEqual, 2)
							So(messages[1].Text, ShouldEqual, text2)
						})
					})

				})
				Convey("Add new message", func() {
					text3 := "hehehe"
					mOrigin, mDestination := models.NewMessagePair(db, origin, destination, photo, text3)
					So(db.AddMessage(mOrigin), ShouldBeNil)
					So(db.AddMessage(mDestination), ShouldBeNil)
					Integrity(db, uOrigin)
					Integrity(db, uDestination)
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
							Convey("First message should be second", func() {
								messages, err := db.GetMessagesFromUser(destination, origin, pagination)
								So(err, ShouldBeNil)
								So(len(messages), ShouldEqual, 3)
								So(messages[2].Text, ShouldEqual, text3)
							})
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
			Convey("Read destination", func() {
				m, err := db.GetMessage(idOrigin)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeFalse)
				m, err = db.GetMessage(idDestination)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeFalse)
				So(db.SetReadMessagesFromUser(destination, origin), ShouldBeNil)
				m, err = db.GetMessage(idDestination)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeTrue)
				m, err = db.GetMessage(idOrigin)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeTrue)
				Integrity(db, uOrigin)
				Integrity(db, uDestination)
				Convey("Destination has no new messages", func() {
					n, err := db.GetUnreadCount(destination)
					So(n, ShouldEqual, 0)
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
