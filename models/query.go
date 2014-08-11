package models

import (
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/url"
	"time"
)

const (
	SeasonSummer     = "summer"
	SeasonWinter     = "winter"
	SeasonAutumn     = "autumn"
	SeasonSpring     = "spring"
	SexMale          = "male"
	SexFemale        = "female"
	LocationFormat   = "%f,%f"
	LocationArgument = "location"
	ageMax           = 100
	ageMin           = 18
	growthMax        = 300
	weightMax        = 1000
)

// SearchQuery represents filtering query for users or user-related objects
type SearchQuery struct {
	Sex          string
	Seasons      []string
	Destinations []string
	AgeMin       int
	AgeMax       int
	WeightMin    int
	WeightMax    int
	GrowthMin    int
	GrowthMax    int
	City         string
	Country      string
	Text         string
	Avatar       string
	Name         string
	Geo          string
	Location     string
	Sort         string
	Sponsor      string
	Host         string
}

// NewQuery returns query object with parsed fields from url params
func NewQuery(q url.Values) (*SearchQuery, error) {
	query := &SearchQuery{}
	return query, mapToStruct(q, query)
}

// ToBson generates mongo query from SearchQuery
func (q *SearchQuery) ToBson() bson.M {
	query := []bson.M{}
	if q.Sex != "" && (q.Sex == SexMale || q.Sex == SexFemale) {
		query = append(query, bson.M{"sex": q.Sex})
	}
	if len(q.Seasons) > 0 {
		seasonsOk := true
		for _, season := range q.Seasons {
			if !(season == SeasonWinter || season == SeasonSpring || season == SeasonAutumn || season == SeasonSummer) {
				seasonsOk = false
			}
		}
		if seasonsOk {
			query = append(query, bson.M{"seasons": bson.M{"$in": q.Seasons}})
		}
	}

	if len(q.Destinations) > 0 {
		query = append(query, bson.M{"destination": bson.M{"$in": q.Destinations}})
	}

	if q.AgeMax == 0 {
		q.AgeMax = ageMax
	}

	if q.AgeMin == 0 {
		q.AgeMin = ageMin
	}

	if q.AgeMin != ageMin || q.AgeMax != ageMax {
		now := time.Now()
		tMax := now.AddDate(-q.AgeMax, 0, 0)
		tMin := now.AddDate(-q.AgeMin, 0, 0)
		query = append(query, bson.M{"birthday": bson.M{"$gte": tMax, "$lte": tMin}})
	}

	if q.GrowthMax == 0 {
		q.GrowthMax = growthMax
	}

	if q.GrowthMax != growthMax || q.GrowthMin != 0 {
		query = append(query, bson.M{"growth": bson.M{"$gte": q.GrowthMin, "$lte": q.GrowthMax}})
	}

	if q.WeightMax == 0 {
		q.WeightMax = weightMax
	}

	if q.WeightMax != weightMax || q.WeightMin != 0 {
		query = append(query, bson.M{"weight": bson.M{"$gte": q.WeightMin, "$lte": q.WeightMax}})
	}

	if q.City != "" {
		query = append(query, bson.M{"city": q.City})
	}

	if q.Country != "" && q.City == "" {
		query = append(query, bson.M{"country": q.Country})
	}
	if q.Avatar != "" {
		query = append(query, bson.M{"avatar": bson.M{"$exists": true}})
	}
	if q.Name != "" {
		pattern := bson.RegEx{Pattern: fmt.Sprintf("^%s", q.Name)}
		query = append(query, bson.M{"name": pattern})
	}

	if q.Geo != "" {
		location := make([]float64, 2)
		_, err := fmt.Sscanf(q.Location, LocationFormat, &location[0], &location[1])
		if err != nil {
			log.Println(err)
		} else {
			geoQuery := bson.M{"location": bson.M{"$near": location}}
			query = append(query, geoQuery)
		}
	}

	if q.Text != "" {
		textQuery := bson.M{"$text": bson.M{"$search": q.Text, "$language": "russian"}}
		query = append(query, textQuery)
	}

	if q.Sponsor != "" {
		query = append(query, bson.M{"is_sponsor": true})
	}

	if q.Host != "" {
		query = append(query, bson.M{"is_host": true})
	}

	if len(query) > 0 {
		return bson.M{"$and": query}
	}

	return bson.M{}
}

type Pagination struct {
	Count  int
	Offset int
}
