package main

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"runtime"
	"time"
)

var (
	salt               = "salt"
	projectName        = "poputchiki"
	dbName             = projectName
	collection         = "users"
	guestsCollection   = "guests"
	messagesCollection = "messages"
	statusesCollection = "statuses"
	photoCollection    = "photo"
	albumsCollection   = "albums"
	mongoHost          = "localhost"
	processes          = runtime.NumCPU()
	redisName          = projectName
	redisAddr          = ":6379"
)

func getHash(password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

type Application struct {
	session *mgo.Session
	p       *redis.Pool
	m       *martini.ClassicMartini
}

func newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisAddr)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func NewDatabase(session *mgo.Session) UserDB {
	coll := session.DB(dbName).C(collection)
	gcoll := session.DB(dbName).C(guestsCollection)
	mcoll := session.DB(dbName).C(messagesCollection)
	scoll := session.DB(dbName).C(statusesCollection)
	pcoll := session.DB(dbName).C(photoCollection)
	acoll := session.DB(dbName).C(albumsCollection)
	return &DB{coll, gcoll, mcoll, scoll, pcoll, acoll}
}

func NewApp() *Application {
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		log.Fatal(err)
	}

	runtime.GOMAXPROCS(processes)
	var db UserDB
	var tokenStorage TokenStorage
	var realtime RealtimeInterface
	db = NewDatabase(session)
	p := newPool()
	tokenStorage = &TokenStorageRedis{p}
	realtime = &RealtimeRedis{p, make(map[bson.ObjectId]ReltChannel)}

	m := martini.Classic()

	m.Use(martini.Static("static", martini.StaticOptions{Prefix: "/"}))
	m.Use(JsonEncoder)
	m.Use(TokenWrapper)
	m.Map(db)
	m.Map(tokenStorage)
	m.Map(realtime)

	m.Post("/api/auth/register", Register)
	m.Post("/api/auth/login", Login)
	m.Post("/api/auth/logout", Logout)

	m.Get("/api/user/:id", GetUser)
	m.Patch("/api/user/:id", Update)
	m.Put("/api/user/:id", Update)

	m.Put("/api/user/:id/messages", SendMessage)
	m.Get("/api/user/:id/messages", GetMessagesFromUser)
	m.Delete("/api/message/:id", RemoveMessage)

	m.Post("/api/user/:id/fav", AddToFavorites)
	m.Delete("/api/user/:id/fav", RemoveFromFavorites)
	m.Get("/api/user/:id/fav", GetFavorites)

	m.Post("/api/user/:id/blacklist", AddToBlacklist)
	m.Delete("/api/user/:id/blacklist", RemoveFromBlacklist)

	m.Post("/api/user/:id/guests", AddToGuests)
	m.Get("/api/user/:id/guests", GetGuests)

	m.Post("/api/image", UploadImage)

	m.Get("/api/realtime", realtime.RealtimeHandler)

	a := Application{session, p, m}
	a.InitDatabase()
	return &a
}

func (a *Application) Close() {
	a.session.Close()
	a.p.Close()
}

func (a *Application) Run() {
	a.m.Run()
}

func (a *Application) DropDatabase() {
	a.session.DB(dbName).C(collection).DropCollection()
	a.session.DB(dbName).C(messagesCollection).DropCollection()
	a.session.DB(dbName).C(guestsCollection).DropCollection()
	a.InitDatabase()
}

func (a *Application) InitDatabase() {
	index := mgo.Index{
		Key:        []string{"email"},
		Background: true, // See notes.
	}
	db := a.session.DB(dbName)
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

	index = mgo.Index{
		Key:        []string{"time"},
		Background: true,
	}
	db.C(messagesCollection).EnsureIndex(index)
	db.C(statusesCollection).EnsureIndex(index)
}

func (a *Application) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	a.m.ServeHTTP(res, req)
}

func main() {
	a := NewApp()
	defer a.Close()
	a.Run()
}
