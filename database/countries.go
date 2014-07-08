package database

import (
	"fmt"
	"labix.org/v2/mgo/bson"
	"sort"
)

func (db *DB) GetCountries(start string) (countries []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", start), Options: "i"}
	query := bson.M{"title": pattern}
	err = db.countries.Find(query).Distinct("title", &countries)
	sort.Strings(countries)
	return countries, err
}

func (db *DB) GetCities(start, country string) (cities []string, err error) {
	pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", start), Options: "i"}
	query := bson.M{"title": pattern, "country": country}
	err = db.cities.Find(query).Distinct("title", &cities)
	sort.Strings(cities)
	return cities, err
}
