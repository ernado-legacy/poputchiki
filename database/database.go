package database

import (
	. "github.com/ernado/poputchiki/models"
	"gopkg.in/mgo.v2"
	"log"
	"time"
)

const (
	stripeCount = 20
	searchCount = 40
)

var (
	collection              = "users"
	citiesCollection        = "cities"
	countriesCollection     = "countries"
	guestsCollection        = "guests"
	messagesCollection      = "messages"
	statusesCollection      = "statuses"
	photoCollection         = "photo"
	albumsCollection        = "albums"
	filesCollection         = "files"
	videoCollection         = "video"
	conftokensCollection    = "conftokens"
	audioCollection         = "audio"
	stripeCollection        = "stripe"
	tokenCollection         = "tokens"
	updatesCollection       = "updates"
	activitiesCollection    = "activities"
	presentsCollection      = "presents"
	adsCollection           = "advertisements"
	presentEventsCollection = "present_events"
)

type DB struct {
	db             *mgo.Database
	users          *mgo.Collection
	guests         *mgo.Collection
	messages       *mgo.Collection
	statuses       *mgo.Collection
	photo          *mgo.Collection
	files          *mgo.Collection
	video          *mgo.Collection
	audio          *mgo.Collection
	stripe         *mgo.Collection
	conftokens     *mgo.Collection
	cities         *mgo.Collection
	countries      *mgo.Collection
	activities     *mgo.Collection
	updates        *mgo.Collection
	presents       *mgo.Collection
	presentEvents  *mgo.Collection
	advertisements *mgo.Collection
	salt           string
	offlineTimeout time.Duration
}

func TestDatabase() *DB {
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatal(err)
	}
	return New("test", "test", time.Second*5, session)
}

// Drop all collections of database
func (db *DB) Drop() {
	collections := []*mgo.Collection{db.users, db.guests, db.messages, db.statuses, db.photo,
		db.files, db.video, db.audio, db.stripe, db.conftokens, db.activities, db.updates, db.presents, db.presentEvents, db.advertisements}

	for k := range collections {
		collections[k].DropCollection()
	}
}

func must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// Init initiates indexes
func (database *DB) Init() {
	index := mgo.Index{
		Key:        []string{"email"},
		Background: true, // See notes.
	}
	db := database.db
	must(db.C(collection).EnsureIndex(index))

	index = mgo.Index{Key: []string{"$hashed:_id"}}
	must(db.C(collection).EnsureIndex(index))
	must(db.C(collection).EnsureIndexKey("rating"))

	index = mgo.Index{
		Key:        []string{"guest"},
		Background: true, // See notes.
	}
	must(db.C(guestsCollection).EnsureIndex(index))

	// photo, guest, messages: hashed user index
	index = mgo.Index{
		Key: []string{"$hashed:user"},
	}
	must(db.C(messagesCollection).EnsureIndex(index))
	must(db.C(guestsCollection).EnsureIndex(index))
	must(db.C(photoCollection).EnsureIndex(index))
	must(db.C(statusesCollection).EnsureIndex(index))
	must(db.C(guestsCollection).EnsureIndex(index))
	must(db.C(filesCollection).EnsureIndex(index))

	index = mgo.Index{
		Key:        []string{"time"},
		Background: true,
	}
	must(db.C(messagesCollection).EnsureIndex(index))
	must(db.C(statusesCollection).EnsureIndex(index))
	must(db.C(filesCollection).EnsureIndex(index))
	must(db.C(stripeCollection).EnsureIndex(index))

	index = mgo.Index{Key: []string{"read", "time", "destination", "type"}}
	must(db.C(updatesCollection).EnsureIndex(index))
	index = mgo.Index{Key: []string{"online", "lastaction"}}
	must(db.C(collection).EnsureIndex(index))

	// index = mgo.Index{Key: []string{"title", "country"}, Unique: true, DropDups: true}
	// must(db.C(citiesCollection).EnsureIndex(index))
	// must(db.C(citiesCollection).EnsureIndexKey("title"))
	// must(db.C(countriesCollection).EnsureIndexKey("title"))

	index = mgo.Index{
		Key: []string{"$2d:location"},
	}
	must(db.C(collection).EnsureIndex(index))
	must(db.C(collection).EnsureIndexKey("$text:status"))
	must(db.C(updatesCollection).EnsureIndexKey("destination"))
	index = mgo.Index{Key: []string{"type", "time", "destination"}}
	must(db.C(presentEventsCollection).EnsureIndex(index))
	must(db.C(presentsCollection).EnsureIndexKey("title"))
}

func New(name, salt string, timeout time.Duration, session *mgo.Session) *DB {
	db := session.DB(name)
	cityDB := session.DB("countries")
	database := new(DB)
	database.db = db
	database.offlineTimeout = timeout
	database.salt = salt
	database.activities = db.C(activitiesCollection)
	database.users = db.C(collection)
	database.guests = db.C(guestsCollection)
	database.messages = db.C(messagesCollection)
	database.statuses = db.C(statusesCollection)
	database.photo = db.C(photoCollection)
	database.files = db.C(filesCollection)
	database.video = db.C(videoCollection)
	database.audio = db.C(audioCollection)
	database.stripe = db.C(stripeCollection)
	database.conftokens = db.C(conftokensCollection)
	database.updates = db.C(updatesCollection)
	database.countries = cityDB.C(countriesCollection)
	database.cities = cityDB.C(citiesCollection)
	database.presents = db.C(presentsCollection)
	database.presentEvents = db.C(presentEventsCollection)
	database.advertisements = db.C(adsCollection)
	database.Init()
	return database
}

func (db *DB) Salt() string {
	return db.salt
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}
