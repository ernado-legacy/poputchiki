package database

import (
	"fmt"
	"github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2/bson"
	"sort"
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

func (db *DB) GetCountries(start string) (countries []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	query := bson.M{"title": pattern}
	err = db.countries.Find(query).Distinct("title", &countries)
	sort.Strings(countries)
	return countries, err
}

func (db *DB) CountryExists(name string) bool {
	count, err := db.countries.Find(bson.M{"title": name}).Count()
	return err == nil && count > 0
}

func (db *DB) CityExists(name string) bool {
	count, err := db.cities.Find(bson.M{"title": name}).Count()
	return err == nil && count > 0
}

func (db *DB) GetCities(start, country string) (cities []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	query := bson.M{"title": pattern, "country": country}
	err = db.cities.Find(query).Distinct("title", &cities)
	sort.Strings(cities)
	return cities, err
}

func (db *DB) GetPlaces(start string) (places []string, err error) {
	var cities []string
	var countries []string
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	query := bson.M{"title": pattern}
	if err = db.cities.Find(query).Distinct("title", &cities); err != nil {
		return
	}
	if err = db.countries.Find(query).Distinct("title", &countries); err != nil {
		return
	}
	sort.Strings(cities)
	sort.Strings(countries)
	places = append(places, countries...)
	places = append(places, cities...)
	return places, err
}

func (db *DB) GetCityPairs(start string) (cities []models.City, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", capitalize(start))}
	return cities, db.cities.Find(bson.M{"title": pattern}).All(&cities)
}
