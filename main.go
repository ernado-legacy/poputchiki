package main

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/gzip"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"runtime"
)

var (
	salt               = "salt"
	projectName        = "poputchiki"
	dbName             = projectName
	collection         = "users"
	guestsCollection   = "guests"
	messagesCollection = "messages"
	mongoHost          = "localhost"
	processes          = runtime.NumCPU()
	redisName          = projectName
)

func getHash(password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

type Application struct {
	session *mgo.Session
	c       redis.Conn
	m       *martini.ClassicMartini
}

func NewApp() *Application {
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		log.Fatal(err)
	}

	c, err := redis.Dial("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}

	runtime.GOMAXPROCS(processes)
	var db UserDB
	var tokenStorage TokenStorage
	coll := session.DB(dbName).C(collection)
	gcoll := session.DB(dbName).C(guestsCollection)
	mcoll := session.DB(dbName).C(messagesCollection)
	db = &DB{coll, gcoll, mcoll}
	tokenStorage = &TokenStorageRedis{c}

	m := martini.Classic()

	m.Use(JsonEncoder)
	m.Use(TokenWrapper)
	m.Use(gzip.All())
	m.Map(db)
	m.Map(c)
	m.Map(tokenStorage)

	m.Post("/api/auth/login", Login)
	m.Post("/api/auth/register", Register)
	m.Post("/api/auth/logout", Logout)

	m.Get("/api/user/:id", GetUser)
	m.Patch("/api/user/:id", Update)

	m.Post("/api/user/:id/fav/", AddToFavorites)
	m.Delete("/api/user/:id/fav/", RemoveFromFavorites)
	m.Get("/api/user/:id/fav/", GetFavorites)

	m.Post("/api/user/:id/guests/", AddToGuests)
	m.Get("/api/user/:id/guests/", GetGuests)

	a := Application{session, c, m}
	a.InitDatabase()
	return &a
}

func (a *Application) Close() {
	a.session.Close()
	a.c.Close()
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
		Unique:     true,
		Background: true, // See notes.
		DropDups:   false,
	}
	a.session.DB(dbName).C(collection).EnsureIndex(index)

	index = mgo.Index{
		Key:        []string{"user", "guest"},
		Unique:     true,
		Background: true, // See notes.
		DropDups:   true,
	}
	a.session.DB(dbName).C(guestsCollection).EnsureIndex(index)

	index = mgo.Index{
		Key:        []string{"user", "origin", "destination", "time"},
		Background: true,
	}
	a.session.DB(dbName).C(messagesCollection).EnsureIndex(index)
}

func (a *Application) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	a.m.ServeHTTP(res, req)
}

func main() {
	a := NewApp()
	defer a.Close()
	a.Run()
}
