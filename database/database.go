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
	db             *mgo.Database
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

// Drop all collections of database
func (db *DB) Drop() {
	collections := []*mgo.Collection{db.users, db.guests, db.messages, db.statuses, db.photo,
		db.albums, db.files, db.video, db.audio, db.stripe, db.conftokens}

	for k := range collections {
		collections[k].DropCollection()
	}
}

// Init initiates indexes
func (database *DB) Init() {
	index := mgo.Index{
		Key:        []string{"email"},
		Background: true, // See notes.
	}
	db := database.db
	db.C(collection).EnsureIndex(index)

	index = mgo.Index{Key: []string{"$hashed:_id"}}
	db.C(collection).EnsureIndex(index)

	index = mgo.Index{
		Key:        []string{"guest"},
		Background: true, // See notes.
	}
	db.C(guestsCollection).EnsureIndex(index)

	// photo, guest, messages: hashed user index
	index = mgo.Index{
		Key: []string{"$hashed:user"},
	}
	db.C(messagesCollection).EnsureIndex(index)
	db.C(guestsCollection).EnsureIndex(index)
	db.C(photoCollection).EnsureIndex(index)
	db.C(statusesCollection).EnsureIndex(index)
	db.C(guestsCollection).EnsureIndex(index)
	db.C(filesCollection).EnsureIndex(index)

	index = mgo.Index{
		Key:        []string{"time"},
		Background: true,
	}
	db.C(messagesCollection).EnsureIndex(index)
	db.C(statusesCollection).EnsureIndex(index)
	db.C(filesCollection).EnsureIndex(index)
	db.C(stripeCollection).EnsureIndex(index)

	index = mgo.Index{Key: []string{"online", "lastaction"}}
	db.C(collection).EnsureIndex(index)
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
	return &DB{db, coll, gcoll, mcoll, scoll, pcoll, acoll, fcoll, vcoll, aucoll,
		stcoll, ctcoll, salt, timeout}
}

func (db *DB) Salt() string {
	return db.salt
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}
