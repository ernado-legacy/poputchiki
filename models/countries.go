package models

type Country struct {
	Id       int    `bson:"_id"`
	Title    string `bson:"title"`
	Priority int    `json:"-"       bson:"priority,omitempty"`
}

type Countries []Country
type Cities []City

func (c Countries) Titles() (titles []string) {
	for i := range c {
		titles = append(titles, c[i].Title)
	}
	return
}

func (c Cities) Titles() (titles []string) {
	for i := range c {
		titles = append(titles, c[i].Title)
	}
	return
}

type City struct {
	Id       int64  `json:"-"       bson:"_id"`
	Title    string `json:"title"   bson:"title"`
	Country  string `json:"country" bson:"country"`
	Priority int    `json:"-"       bson:"priority,omitempty"`
}
