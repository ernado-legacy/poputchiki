package database

import (
	. "github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"time"
)

const (
	stripeCount = 20
	searchCount = 40
)

var (
	collection           = "users"
	citiesCollection     = "cities"
	countriesCollection  = "countries"
	guestsCollection     = "guests"
	messagesCollection   = "messages"
	statusesCollection   = "statuses"
	photoCollection      = "photo"
	albumsCollection     = "albums"
	filesCollection      = "files"
	videoCollection      = "audio"
	conftokensCollection = "conftokens"
	audioCollection      = "video"
	stripeCollection     = "stripe"
	tokenCollection      = "tokens"
)

type DB struct {
	users          *mgo.Collection
	guests         *mgo.Collection
	messages       *mgo.Collection
	statuses       *mgo.Collection
	photo          *mgo.Collection
	albums         *mgo.Collection
	files          *mgo.Collection
	video          *mgo.Collection
	audio          *mgo.Collection
	stripe         *mgo.Collection
	conftokens     *mgo.Collection
	salt           string
	offlineTimeout time.Duration
}

func New(name, salt string, timeout time.Duration, session *mgo.Session) *DB {
	db := session.DB(name)
	coll := db.C(collection)
	gcoll := db.C(guestsCollection)
	mcoll := db.C(messagesCollection)
	scoll := db.C(statusesCollection)
	pcoll := db.C(photoCollection)
	acoll := db.C(albumsCollection)
	fcoll := db.C(filesCollection)
	vcoll := db.C(videoCollection)
	aucoll := db.C(audioCollection)
	stcoll := db.C(stripeCollection)
	ctcoll := db.C(conftokensCollection)
	return &DB{coll, gcoll, mcoll, scoll, pcoll, acoll, fcoll, vcoll, aucoll, stcoll, ctcoll, salt, timeout}
}

func (db *DB) Salt() string {
	return db.salt
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}
