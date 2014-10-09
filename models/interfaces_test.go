package models

import (
	. "github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
)

func preparable(val interface{}) bool {
	_, ok := val.(Preparable)
	return ok
}

func ShouldPrepare(val interface{}) {
	t := reflect.TypeOf(val)
	name := t.Elem().Name()
	if len(name) == 0 {
		name = t.Name()
	}
	Convey(name, func() {
		So(preparable(val), ShouldBeTrue)
	})
}

func TestImplementation(t *testing.T) {
	Convey("Prepare implementation", t, func() {
		ShouldPrepare(&User{})
		ShouldPrepare(Users{})
		ShouldPrepare(Guests{})
		ShouldPrepare(&Photo{})
		ShouldPrepare(&Video{})
		ShouldPrepare(&Audio{})
		ShouldPrepare(PhotoSlice{})
		ShouldPrepare(VideoSlice{})
		ShouldPrepare(&Status{})
		ShouldPrepare(&Message{})
		ShouldPrepare(&StripeItem{})
		ShouldPrepare(&Update{})
	})
}
