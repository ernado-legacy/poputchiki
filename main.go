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
	weedHost           = "localhost"
	weedPort           = 9333
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
	db := session.DB(dbName)
	coll := db.C(collection)
	gcoll := db.C(guestsCollection)
	mcoll := db.C(messagesCollection)
	scoll := db.C(statusesCollection)
	pcoll := db.C(photoCollection)
	acoll := db.C(albumsCollection)
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
	m.Group("/api/auth", func(r martini.Router) {
		r.Post("/register", Register)
		r.Post("/login", Login)
		r.Post("/logout", NeedAuth, Logout)
	})
	m.Group("/api", func(r martini.Router) {
		r.Group("/user/:id", func(r martini.Router) {
			r.Get("/", GetUser)
			r.Patch("/", Update)
			r.Put("/", Update)

			r.Get("/status", GetCurrentStatus)

			r.Put("/messages", SendMessage)
			r.Get("/messages", GetMessagesFromUser)

			r.Post("/fav", AddToFavorites)
			r.Delete("/fav", RemoveFromFavorites)
			r.Get("/fav", GetFavorites)

			r.Post("/blacklist", AddToBlacklist)
			r.Delete("/blacklist", RemoveFromBlacklist)

			r.Post("/guests", AddToGuests)
			r.Get("/guests", GetGuests)
		}, IdWrapper)

		r.Put("/status", AddStatus)
		r.Group("/status/:id", func(r martini.Router) {
			r.Get("/", GetStatus)
			r.Put("/", UpdateStatus)
			r.Delete("/", RemoveStatus)
		}, IdWrapper)

		r.Delete("/message/:id", IdWrapper, RemoveMessage)
		r.Post("/image", UploadImage)
		r.Get("/realtime", realtime.RealtimeHandler)
	}, NeedAuth)

	a := &Application{session, p, m}
	a.InitDatabase()
	return a
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
