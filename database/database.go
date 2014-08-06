package database

import (
	. "github.com/ernado/poputchiki/models"
	"labix.org/v2/mgo"
	"log"
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
	cities         *mgo.Collection
	countries      *mgo.Collection
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
		db.albums, db.files, db.video, db.audio, db.stripe, db.conftokens}

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

	index = mgo.Index{Key: []string{"online", "lastaction"}}
	must(db.C(collection).EnsureIndex(index))

	// must(db.C(citiesCollection).EnsureIndexKey("title", "country"))
	index = mgo.Index{Key: []string{"title", "country"}, Unique: true}
	must(db.C(citiesCollection).EnsureIndex(index))
	must(db.C(citiesCollection).EnsureIndexKey("title"))
	must(db.C(countriesCollection).EnsureIndexKey("title"))

	index = mgo.Index{
		Key: []string{"$2d:location"},
	}
	must(db.C(collection).EnsureIndex(index))
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
	citb := session.DB("countries")
	cotc := citb.C(countriesCollection)
	citc := citb.C("cities")
	database := &DB{db, coll, gcoll, mcoll, scoll, pcoll, acoll, fcoll, vcoll, aucoll,
		stcoll, ctcoll, citc, cotc, salt, timeout}
	database.Init()
	return database
}

func (db *DB) Salt() string {
	return db.salt
}

func (db *DB) AddFile(file *File) (*File, error) {
	return file, db.files.Insert(file)
}
