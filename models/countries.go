package models

type Country struct {
	Id    int    `bson:"_id"`
	Title string `bson:"title"`
}

type City struct {
	Id      int64  `json:"-"       bson:"_id"`
	Title   string `json:"title"   bson:"title"`
	Country string `json:"country" bson:"country"`
}
