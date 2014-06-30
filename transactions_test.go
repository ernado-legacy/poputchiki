package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"labix.org/v2/mgo/bson"
	"testing"
)

func TestTransactions(t *testing.T) {
	Convey("Handler", t, func() {
		dbName = "poputchiki_transaction_db"
		a := NewApp()
		session := a.session
		redisName = "poputchiki_test_transaction"
		pool := newPool()
		db := session.DB(dbName)
		db.DropDatabase()
		Reset(func() {
			db.DropDatabase()
		})
		handler := NewTransactionHandler(pool, db, "login", "pwd1", "pwd2")
		So(handler.UpdateID(), ShouldBeNil)
		id, err := handler.getID()
		So(err, ShouldBeNil)
		So(id, ShouldEqual, 1)
		id, err = handler.getID()
		So(err, ShouldBeNil)
		So(id, ShouldEqual, 2)
		Convey("New", func() {
			expected := "https://auth.robokassa.ru/Merchant/Index.aspx?Desc=test&InvId=3&MrchLogin=login&OutSum=100&SignatureValue=265254922965146050cd69499acf1e25"
			url, _, err := handler.Start(bson.NewObjectId(), 100, "test")
			So(err, ShouldBeNil)
			So(expected, ShouldEqual, url)
			Convey("Another", func() {
				So(handler.UpdateID(), ShouldBeNil)
				id, err := handler.getID()
				So(err, ShouldBeNil)
				So(id, ShouldEqual, 4)
			})
		})

	})
}
