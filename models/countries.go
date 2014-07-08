package models

type Country struct {
	Id    int    `bson:"_id"`
	Title string `bson:"title"`
}

type City struct {
	Id      int64  `bson:"_id"`
	Title   string `bson:"title"`
	Country string `bson:"country"`
}
