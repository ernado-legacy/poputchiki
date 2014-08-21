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
	Convey("CityPairs", t, func() {
		pairs, err := db.GetCityPairs("моск")
		So(err, ShouldBeNil)
		found := false
		for _, pair := range pairs {
			if pair.Country == "Россия" && pair.Title == "Москва" {
				found = true
				break
			}
		}
		So(found, ShouldBeTrue)
	})
	Convey("Places", t, func() {
		places, err := db.GetPlaces("Росси")
		So(err, ShouldBeNil)
		So(places, ShouldContain, "Россия")
	})
	Convey("Existance", t, func() {
		Convey("Country", func() {
			Convey("Positive", func() {
				So(db.CountryExists("Россия"), ShouldBeTrue)
			})
			Convey("Negative", func() {
				So(db.CountryExists("Абырвалн"), ShouldBeFalse)
			})
			Convey("False positive", func() {
				So(db.CountryExists("россия"), ShouldBeFalse)
			})
		})
		Convey("City", func() {
			Convey("Positive", func() {
				So(db.CityExists("Москва"), ShouldBeTrue)
			})
			Convey("Negative", func() {
				So(db.CityExists("Абырвалн"), ShouldBeFalse)
			})
			Convey("False positive", func() {
				So(db.CityExists("москва"), ShouldBeFalse)
			})
		})
	})
}
