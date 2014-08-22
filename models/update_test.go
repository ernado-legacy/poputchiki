package models

import (
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestUpdates(t *testing.T) {
	Convey("Make update", t, func() {
		destination := bson.NewObjectId()
		user := bson.NewObjectId()
		Convey("Photo like", func() {
			p := new(Photo)
			u := NewUpdate(destination, user, UpdateLikes, p)
			So(u.TargetType, ShouldEqual, "photo")
		})
		Convey("Message", func() {
			p := new(Message)
			u := NewUpdate(destination, user, UpdateMessages, p)
			So(u.TargetType, ShouldEqual, "message")
		})
	})

	Convey("Update types", t, func() {
		Convey("Photo", func() {
			p := new(Photo)
			So(GetEventType(UpdateLikes, p), ShouldEqual, SubscriptionLikesPhoto)
		})
		Convey("Status", func() {
			p := new(Status)
			So(GetEventType(UpdateLikes, p), ShouldEqual, SubscriptionLikesStatus)
		})
		Convey("Guests", func() {
			So(GetEventType(UpdateGuests, nil), ShouldEqual, SubscriptionGuests)
		})

		Convey("Messages", func() {
			m := new(Message)
			So(GetEventType(UpdateMessages, m), ShouldEqual, SubscriptionMessages)
		})
	})
}
