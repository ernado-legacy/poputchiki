package database

import (
	"testing"
	"time"

	"github.com/ernado/poputchiki/activities"
	"github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
)

func TestActivities(t *testing.T) {
	db := TestDatabase()
	Convey("Activities", t, func() {
		Reset(db.Drop)
		id := bson.NewObjectId()
		u := &models.User{Id: id, Email: "dasdsadas@asdasd", Sex: models.SexFemale, Name: "Kakesh Malakesh"}
		So(db.Add(u), ShouldBeNil)
		Integrity(db, u)
		Convey("Add", func() {
			addCount := 10
			for i := 0; i < addCount; i++ {
				So(db.AddActivity(id, activities.Invite), ShouldBeNil)
			}
			Integrity(db, u)
			Convey("Count", func() {
				count, err := db.GetActivityCount(id, activities.Invite, time.Hour)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, addCount)
			})
		})
	})

}
