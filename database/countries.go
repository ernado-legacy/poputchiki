package database

import (
	"fmt"
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
	"unicode"
)

func capitalize(s string) string {
	if len(s) < 1 {
		return s
	}
	a := []rune(s)
	a[0] = unicode.ToUpper(a[0])
	return string(a)
}

func (db *DB) GetCountries(start string) (titles []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	query := bson.M{"title": pattern}
	countries := new(models.Countries)
	err = db.countries.Find(query).Sort("title").Sort("-priority").Limit(100).All(countries)
	return countries.Titles(), err
}

func (db *DB) CountryExists(name string) bool {
	count, err := db.countries.Find(bson.M{"title": name}).Count()
	return err == nil && count > 0
}

func (db *DB) CityExists(name string) bool {
	count, err := db.cities.Find(bson.M{"title": name}).Count()
	return err == nil && count > 0
}

func (db *DB) GetCities(start, country string) (titles []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	query := bson.M{"title": pattern, "country": country}
	cities := new(models.Cities)
	err = db.cities.Find(query).Sort("title").Sort("-priority").Limit(100).All(cities)
	return cities.Titles(), err
}

func (db *DB) GetPlaces(start string) (places []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	query := bson.M{"title": pattern}
	cities := new(models.Cities)
	countries := new(models.Countries)
	if err = db.cities.Find(query).Sort("title").Sort("-priority").Limit(100).All(cities); err != nil {
		return
	}
	if err = db.countries.Find(query).Sort("title").Sort("-priority").Limit(100).All(countries); err != nil {
		return
	}
	places = append(places, countries.Titles()...)
	places = append(places, cities.Titles()...)
	return places, err
}

func (db *DB) GetCityPairs(start string) (cities models.Cities, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	return cities, db.cities.Find(bson.M{"title": pattern}).Sort("title").Sort("-priority").Limit(100).All(&cities)
}
