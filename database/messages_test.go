package database

import (
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"log"
	"testing"
)

func TestMessages(t *testing.T) {
	db := TestDatabase()
	origin := bson.NewObjectId()
	destination := bson.NewObjectId()
	text := "Привет"
	pagination := models.Pagination{}
	Convey("Add users", t, func() {
		Reset(db.Drop)
		uOrigin := &models.User{Id: origin, Name: "Alex"}
		uDestination := &models.User{Id: destination, Name: "Petya"}
		So(db.Add(uOrigin), ShouldBeNil)
		So(db.Add(uDestination), ShouldBeNil)
		Convey("Integrity", Integrity(db, uOrigin))
		Convey("Integrity", Integrity(db, uDestination))
		Convey("Add message", func() {
			mOrigin, mDestination := models.NewMessagePair(db, origin, destination, text)
			idOrigin := mOrigin.Id
			idDestination := mDestination.Id
			So(db.AddMessage(mOrigin), ShouldBeNil)
			So(db.AddMessage(mDestination), ShouldBeNil)
			Convey("Integrity", Integrity(db, uOrigin))
			Convey("Integrity", Integrity(db, uDestination))
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
				mOrigin, mDestination := models.NewMessagePair(db, origin, destination, text2)
				So(db.AddMessage(mOrigin), ShouldBeNil)
				So(db.AddMessage(mDestination), ShouldBeNil)
				Convey("Integrity", Integrity(db, uOrigin))
				Convey("Integrity", Integrity(db, uDestination))
				Convey("Set read origin", func() {
					So(db.SetReadMessagesFromUser(destination, origin), ShouldBeNil)
					Convey("Integrity", Integrity(db, uOrigin))
					Convey("Integrity", Integrity(db, uDestination))
					Convey("Destination has no new messages", func() {
						n, err := db.GetUnreadCount(destination)
						So(n, ShouldEqual, 0)
						So(err, ShouldBeNil)
					})
				})
				Convey("Remove", func() {
					So(db.RemoveMessage(idOrigin), ShouldBeNil)
					Convey("Integrity", Integrity(db, uOrigin))
					Convey("Integrity", Integrity(db, uDestination))
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
					mOrigin, mDestination := models.NewMessagePair(db, origin, destination, text3)
					So(db.AddMessage(mOrigin), ShouldBeNil)
					So(db.AddMessage(mDestination), ShouldBeNil)
					Convey("Integrity", Integrity(db, uOrigin))
					Convey("Integrity", Integrity(db, uDestination))
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
				So(db.SetRead(destination, idDestination), ShouldBeNil)
				m, err := db.GetMessage(idDestination)
				So(err, ShouldBeNil)
				So(m.Read, ShouldBeTrue)
				Convey("Integrity", Integrity(db, uOrigin))
				Convey("Integrity", Integrity(db, uDestination))
				Convey("Destination has no new messages", func() {
					n, err := db.GetUnreadCount(destination)
					So(n, ShouldEqual, 0)
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
