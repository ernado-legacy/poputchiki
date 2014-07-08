package database

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCountries(t *testing.T) {
	db := TestDatabase()
	Convey("Find Russia", t, func() {
		countries, err := db.GetCountries("Росс")
		So(err, ShouldBeNil)
		So(countries, ShouldContain, "Россия")
	})

	Convey("Find Moscow", t, func() {
		cities, err := db.GetCities("Моск", "Россия")
		So(err, ShouldBeNil)
		So(cities, ShouldContain, "Москва")
	})
	Convey("Case insensitive", t, func() {
		cities, err := db.GetCities("моск", "Россия")
		So(err, ShouldBeNil)
		So(cities, ShouldContain, "Москва")
		countries, err := db.GetCountries("рос")
		So(err, ShouldBeNil)
		So(countries, ShouldContain, "Россия")
	})
}
