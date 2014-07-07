package models

import (
	"encoding/json"
	"labix.org/v2/mgo/bson"
	"log"
	"net/url"
	"strconv"
	"time"
)

const (
	SEASON_SUMMER = "summer"
	SEASON_WINTER = "winter"
	SEASON_AUTUMN = "autumn"
	SEASON_SPRING = "spring"
	sexMale       = "male"
	sexFemale     = "female"
	ageMax        = 100
	ageMin        = 18
	growthMax     = 300
	weightMax     = 1000
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
}

// NewQuery returns query object with parsed field from url params
func NewQuery(q url.Values) (*SearchQuery, error) {
	nQ := make(map[string]interface{})
	for key, value := range q {
		if len(value) == 1 {
			v := value[0]
			vInt, err := strconv.Atoi(v)
			if err != nil {
				nQ[key] = v
			} else {
				nQ[key] = vInt
			}
		} else {
			nQ[key] = value
		}
	}
	j, err := json.Marshal(nQ)
	if err != nil {
		return nil, err
	}
	query := &SearchQuery{}
	err = json.Unmarshal(j, query)
	if err != nil {
		return nil, err
	}
	return query, nil
}

// ToBson converts search query to bson map
func (q *SearchQuery) ToBson() bson.M {
	query := []bson.M{}
	if q.Sex != BLANK && (q.Sex == sexMale || q.Sex == sexFemale) {
		query = append(query, bson.M{"sex": q.Sex})
	}
	if len(q.Seasons) > 0 {
		query = append(query, bson.M{"seasons": bson.M{"$in": q.Seasons}})
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

	if q.City != BLANK {
		query = append(query, bson.M{"city": q.City})
	}

	if q.Country != BLANK && q.City == BLANK {
		query = append(query, bson.M{"country": q.Country})
	}

	fullQuery := bson.M{"$and": query}
	m, _ := json.Marshal(fullQuery)
	log.Println(string(m))
	return fullQuery
}

type Pagination struct {
	Count  int
	Offset int
}
