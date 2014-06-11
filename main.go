package main

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"github.com/coreos/go-etcd/etcd"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"runtime"
	"strconv"
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
	filesCollection    = "files"
	videoCollection    = "audio"
	audioCollection    = "video"
	stripeCollection   = "stripe"
	mongoHost          = "localhost"
	processes          = runtime.NumCPU()
	redisName          = projectName
	redisAddr          = ":6379"
	weedHost           = "msk1.cydev.ru"
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

func loadConfig() error {
	c := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	_, err := c.Update("test", "test", 0)
	if err != nil {
		return err
	}
	mongo, err := c.Get("mongodb-master", false, false)
	if err != nil {
		return err
	}
	mongoHost = mongo.Node.Value
	redisMaster, err := c.Get("redis-master", false, false)
	if err != nil {
		return err
	}
	redisAddr = redisMaster.Node.Value
	weedMaster, err := c.Get("weed-master/host", false, false)
	if err != nil {
		return err
	}
	weedHost = weedMaster.Node.Value
	weedMaster, err = c.Get("weed-master/port", false, false)
	if err != nil {
		return err
	}
	weedPort, err = strconv.Atoi(weedMaster.Node.Value)
	return err
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
	fcoll := db.C(filesCollection)
	vcoll := db.C(videoCollection)
	aucoll := db.C(audioCollection)
	stcoll := db.C(stripeCollection)
	return &DB{coll, gcoll, mcoll, scoll, pcoll, acoll, fcoll, vcoll, aucoll, stcoll}
}

func DataBase() martini.Handler {
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		log.Fatal(err)
	}

	return func(c martini.Context) {
		var db UserDB
		s := session.Clone()
		db = NewDatabase(s)
		// defer s.Close()
		c.Map(db)
		c.Next()
	}
}

func NewApp() *Application {
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		log.Fatal(err)
	}

	runtime.GOMAXPROCS(processes)
	// var db UserDB
	var tokenStorage TokenStorage
	var realtime RealtimeInterface
	// db = NewDatabase(session)
	p := newPool()
	tokenStorage = &TokenStorageRedis{p}
	realtime = &RealtimeRedis{p, make(map[bson.ObjectId]ReltChannel)}

	m := martini.Classic()

	m.Use(JsonEncoder)
	m.Use(JsonEncoderWrapper)
	m.Use(TokenWrapper)
	m.Use(WebpWrapper)
	m.Use(AudioWrapper)
	m.Use(VideoWrapper)
	m.Use(PaginationWrapper)
	m.Use(DataBase())
	m.Map(tokenStorage)
	m.Map(realtime)
	m.Group("/api/auth", func(r martini.Router) {
		r.Post("/register", Register)
		r.Post("/login", Login)
		r.Post("/logout", NeedAuth, Logout)
	})
	m.Group("/api", func(r martini.Router) {
		r.Group("/user/:id", func(r martini.Router) {
			r.Get("", GetUser)
			r.Get("/status", GetCurrentStatus)

			r.Put("/messages", SendMessage)
			r.Get("/messages", GetMessagesFromUser)

			r.Group("", func(d martini.Router) {
				d.Patch("", Update)
				d.Put("", Update)
				d.Post("/fav", AddToFavorites)
				d.Delete("/fav", RemoveFromFavorites)
				d.Get("/fav", GetFavorites)

				d.Post("/blacklist", AddToBlacklist)
				d.Delete("/blacklist", RemoveFromBlacklist)

				d.Post("/guests", AddToGuests)
				d.Get("/guests", GetGuests)
			}, IdEqualityRequired)

		}, IdWrapper)

		r.Put("/status", AddStatus)
		r.Group("/status/:id", func(r martini.Router) {
			r.Get("", GetStatus)
			r.Put("", UpdateStatus)
			r.Delete("", RemoveStatus)
		}, IdWrapper)

		r.Delete("/message/:id", IdWrapper, RemoveMessage)
		r.Post("/album/:id/photo", IdWrapper, UploadPhotoToAlbum)
		r.Put("/album", AddAlbum)
		r.Post("/video", UploadVideo)
		r.Post("/photo", UploadPhoto)
		r.Get("/realtime", realtime.RealtimeHandler)
		r.Get("/search", SearchPeople)
	}, NeedAuth)

	a := &Application{session, p, m}
	a.InitDatabase()
	return a
}

func (a *Application) Close() {
	// a.session.Close()
	a.p.Close()
}

func (a *Application) Run() {
	a.m.Run()
}

func (a *Application) DropDatabase() {
	a.session.DB(dbName).C(collection).DropCollection()
	a.session.DB(dbName).C(messagesCollection).DropCollection()
	a.session.DB(dbName).C(guestsCollection).DropCollection()
	a.session.DB(dbName).C(filesCollection).DropCollection()
	a.session.DB(dbName).C(statusesCollection).DropCollection()
	a.session.DB(dbName).C(stripeCollection).DropCollection()
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
	db.C(filesCollection).EnsureIndex(index)

	index = mgo.Index{
		Key:        []string{"time"},
		Background: true,
	}
	db.C(messagesCollection).EnsureIndex(index)
	db.C(statusesCollection).EnsureIndex(index)
	db.C(filesCollection).EnsureIndex(index)
	db.C(stripeCollection).EnsureIndex(index)
}

func (a *Application) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	a.m.ServeHTTP(res, req)
}

func main() {
	a := NewApp()
	saltF := flag.String("salt", "salt", "salt")
	projectNameF := flag.String("name", "poputchiki", "project name")
	mongoHostF := flag.String("mongo", "localhost", "mongo host")
	redisAddrF := flag.String("redis", ":6379", "redis host")
	weedHostF := flag.String("weed", "msk1.cydev.ru", "weed host")
	flag.Parse()
	projectName = *projectNameF
	dbName = projectName
	redisName = projectName
	redisAddr = *redisAddrF
	mongoHost = *mongoHostF
	weedHost = *weedHostF
	salt = *saltF
	defer a.Close()
	a.Run()
}
