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
			mOrigin := &models.Message{idOrigin, destination, origin, origin, destination, false, now, text}
			mDestination := &models.Message{idDestination, origin, destination, origin, destination, false, now, text}
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
		})
	})
}
