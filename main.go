package main

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/ernado/cymedia/mediad/query"
	"github.com/ernado/gofbauth"
	"github.com/ernado/gotok"
	"github.com/ernado/govkauth"
	"github.com/ernado/poputchiki/database"
	"github.com/ernado/poputchiki/models"
	"github.com/ernado/weed"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	// "github.com/martini-contrib/throttle"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"runtime"
	"time"
)

var (
	salt                      = "salt"
	projectName               = "poputchiki"
	premiumTime               = time.Hour * 24 * 30
	statusUpdateTime          = time.Hour * 24
	dbName                    = projectName
	dbCity                    = "countries"
	tokenCollection           = "tokens"
	mongoHost                 = "localhost"
	robokassaLogin            = "poputchiki.ru"
	robokassaPassword1        = "pcZKT5Qm84MJAIudLAbR"
	robokassaPassword2        = "8x3cVXUt08Uc9TV70mx3"
	robokassaDescription      = "Пополнение счета Попутчики.ру"
	production                = false
	processes                 = runtime.NumCPU()
	redisName                 = projectName
	redisAddr                 = ":6379"
	redisQueryKey             = flag.String("query.key", "poputchiki:conventer:query", "Convertation query key")
	mailKey                   = "key-7520cy18i2ebmrrbs1bz4ivhua-ujtb6"
	mailDomain                = "mg.cydev.ru"
	smsKey                    = "nil"
	weedHost                  = "127.0.0.1"
	weedPort                  = 9333
	weedUrl                   = fmt.Sprintf("http://%s:%d", weedHost, weedPort)
	OfflineTimeout            = 60 * 5 * time.Second
	OfflineUpdateTick         = 5 * time.Second
	PromoCost            uint = 50
	mobile                    = flag.Bool("mobile", false, "is mobile api")
	development               = flag.Bool("dev", false, "is in development")
	sendEmail                 = flag.Bool("email", true, "send registration emails")
)

func getHash(password string, s string) string {
	// log.Printf("sha256(%s,%s)", password, s)
	hasher := sha256.New()
	hasher.Write([]byte(password + s))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

type Application struct {
	session *mgo.Session
	p       *redis.Pool
	m       *martini.ClassicMartini
	db      models.DataBase
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

func NewDatabase(session *mgo.Session) models.DataBase {
	return database.New(dbName, salt, OfflineTimeout, session)
}

func NewApp() *Application {
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		log.Fatal(err)
	}
	session.SetMode(mgo.Monotonic, true)
	runtime.GOMAXPROCS(processes)
	var db models.DataBase
	var realtime models.RealtimeInterface
	var tokenStorage gotok.Storage
	db = NewDatabase(session)
	p := newPool()
	tokenStorage = gotok.New(session.DB(dbName).C(tokenCollection))
	realtime = &RealtimeRedis{p, make(map[bson.ObjectId]ReltChannel)}

	m := martini.Classic()

	if production {
		martini.Env = martini.Prod
	}

	// A Rate Limit Policy
	// m.Use(throttle.Policy(&throttle.Quota{
	// 	Limit:  1000,
	// 	Within: time.Hour,
	// }))

	// // An Interval Policy
	// m.Use(throttle.Policy(&throttle.Quota{
	// 	Limit:  10,
	// 	Within: time.Second,
	// }))

	queryClient, err := query.NewRedisClient(weedUrl, redisAddr, *redisQueryKey, redisQueryRespKey)
	if err != nil {
		log.Fatal(err)
	}

	m.Map(queryClient)
	m.Map(&govkauth.Client{"4456019", "0F4CUYU2Iq9H7YhANtdf", "http://poputchiki.ru/api/auth/vk/redirect", "offline,email"})
	m.Map(&gofbauth.Client{"1518821581670594", "97161fd30ed48e5a3e25811ed02d0f3a", "http://poputchiki.ru/api/auth/fb/redirect", "email,user_birthday"})
	m.Use(JsonEncoder)
	m.Use(JsonEncoderWrapper)
	m.Use(TokenWrapper)
	m.Use(WebpWrapper)
	m.Use(AudioWrapper)
	m.Use(VideoWrapper)
	m.Use(PaginationWrapper)
	m.Use(ParserWrapper)
	m.Map(tokenStorage)
	m.Map(realtime)
	m.Map(weed.NewAdapter(weedUrl))
	m.Map(db)
	m.Map(NewTransactionHandler(p, session.DB(dbName), robokassaLogin, robokassaPassword1, robokassaPassword2))

	staticOptions := martini.StaticOptions{Prefix: "/api/static/"}
	m.Use(martini.Static("static", staticOptions))

	root := "/api"
	if *mobile {
		root = "/api/mobile"
	}

	m.Group(root+"/auth", func(r martini.Router) {
		r.Post("/register", Register)
		r.Post("/login", Login)
		r.Get("/login", Login)
		r.Post("/logout", NeedAuth, Logout)
		r.Post("/forgot/:email", ForgotPassword)
		r.Get("/reset/:token", ResetPassword)
		r.Get("/vk/start", VkontakteAuthStart)
		r.Get("/fb/start", FacebookAuthStart)
		r.Get("/vk/redirect", VkontakteAuthRedirect)
		r.Get("/fb/redirect", FacebookAuthRedirect)
	})
	m.Get(root, Index)
	m.Get(root+"/pay/success", RobokassaSuccessHandler)
	m.Group(root, func(r martini.Router) {
		r.Get("/admin", AdminView)
		r.Get("/confirm/email/:token", ConfirmEmail)
		r.Get("/confirm/phone/start", ConfirmPhoneStart)
		r.Get("/confirm/phone/:token", ConfirmPhone)

		r.Post("/pay/:value", GetTransactionUrl)
		r.Get("/pay/:value", GetTransactionUrl)

		r.Get("/token", GetToken)

		r.Group("/user/:id", func(r martini.Router) {
			r.Get("", GetUser)
			r.Get("/status", GetCurrentStatus)
			r.Get("/login", AdminLogin)
			r.Put("/messages", SendMessage)
			r.Get("/messages", GetMessagesFromUser)
			r.Get("/chats", GetChats)
			r.Get("/photo", GetUserPhoto)
			r.Group("", func(d martini.Router) {
				d.Patch("", Update)
				d.Put("", Update)
				d.Post("", Update)

				d.Post("/fav", AddToFavorites)
				d.Put("/fav", AddToFavorites)
				d.Delete("/fav", RemoveFromFavorites)
				d.Get("/fav", GetFavorites)

				d.Post("/blacklist", AddToBlacklist)
				d.Put("/blacklist", AddToBlacklist)
				d.Delete("/blacklist", RemoveFromBlacklist)

				d.Post("/guests", AddToGuests)
				d.Put("/guests", AddToGuests)
				d.Get("/guests", GetGuests)
				d.Get("/unread", GetUnreadCount)

			}, IdEqualityRequired)

		}, IdWrapper)

		r.Get("/stripe", GetStripe)
		r.Post("/stripe", AddStripeItem)
		r.Put("/stripe", AddStripeItem)

		r.Put("/status", AddStatus)
		r.Post("/status", AddStatus)
		r.Get("/status", SearchStatuses)
		r.Group("/status/:id", func(r martini.Router) {
			r.Get("", GetStatus)
			r.Put("", UpdateStatus)
			r.Post("", UpdateStatus)
			r.Delete("", RemoveStatus)
			r.Get("/like", GetLikersStatus)
			r.Post("/like", LikeStatus)
			r.Delete("/like", RestoreLikeStatus)
		}, IdWrapper)
		
		r.Delete("/message/:id", IdWrapper, RemoveMessage)
		r.Post("/message/:id/read", IdWrapper, MarkReadMessage)
		r.Post("/video", UploadVideoFile)
		r.Post("/audio", UploadAudio)
		r.Post("/video/:id/like", IdWrapper, LikeVideo)
		r.Get("/video/:id/like", IdWrapper, GetLikersVideo)
		r.Delete("/video/:id/like", IdWrapper, RestoreLikeVideo)
		r.Post("/photo", UploadPhoto)
		r.Get("/realtime", realtime.RealtimeHandler)
		r.Get("/search", SearchPeople)
		r.Get("/photo", SearchPhoto)
		r.Post("/photo/:id/like", IdWrapper, LikePhoto)
		r.Get("/photo/:id/like", IdWrapper, GetLikersPhoto)
		r.Delete("/photo/:id/like", IdWrapper, RestoreLikePhoto)
		r.Get("/countries", GetCountries)
		r.Get("/cities", GetCities)
		r.Get("/places", GetPlaces)
	}, NeedAuth, SetOnlineWrapper)

	a := &Application{session, p, m, db}
	a.InitDatabase()
	return a
}

func (a *Application) Close() {
	a.p.Close()
}

func (a *Application) StatusCycle() {
	log.Println("[updater]", "starting cycle")
	ticker := time.NewTicker(OfflineUpdateTick)
	for _ = range ticker.C {
		_, err := a.db.UpdateAllStatuses()
		if err != nil {
			log.Println("[updater]", "status update error", err)
			time.Sleep(time.Second * 5)
		}
	}
}

var redisQueryRespKey = "poputchiki:conventer:resp"

func (a *Application) ConvertResultListener() {
	db := a.db
	log.Println("Started conventer", *redisQueryKey, "->", redisQueryRespKey)
	q, err := query.NewRedisResponceQuery(redisAddr, redisQueryRespKey)
	if err != nil {
		log.Println("Unable to create result listener", err)
		return
	}
	for {
		resp, err := q.Pull()
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
		}
		log.Println(resp)
		if !resp.Success {
			log.Println("convertation error", resp.Id, resp.Error)
			continue
		}
		id := bson.ObjectIdHex(resp.Id)
		fid := resp.File
		if resp.Type == "audio" {
			if resp.Format == "ogg" {
				err = db.UpdateAudioOGG(id, fid)
			}
			if resp.Format == "aac" {
				err = db.UpdateAudioAAC(id, fid)
			}
		}
		if resp.Type == "video" {
			if resp.Format == "webm" {
				err = db.UpdateVideoWebm(id, fid)
			}
			if resp.Format == "mp4" {
				err = db.UpdateVideoMpeg(id, fid)
			}
		}
		if err != nil {
			log.Println(resp.Id, err)
		}
	}
}

func (a *Application) Run() {
	go a.StatusCycle()
	go a.ConvertResultListener()
	a.m.Run()
}

func (a *Application) DropDatabase() {
	a.db.Drop()
	a.InitDatabase()
}

func (a *Application) InitDatabase() {
	a.db.Init()
}

func (a *Application) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	a.m.ServeHTTP(res, req)
}

func main() {
	saltF := flag.String("salt", "salt", "salt")
	projectNameF := flag.String("name", "poputchiki", "project name")
	mongoHostF := flag.String("mongo", "localhost", "mongo host")
	redisAddrF := flag.String("redis", ":6379", "redis host")
	weedHostF := flag.String("weed", "127.0.0.1", "weed host")
	flag.BoolVar(&production, "production", false, "environment")
	flag.StringVar(&mailKey, "mail-key", mailKey, "mailgun api key")
	flag.StringVar(&mailDomain, "mail-domain", mailDomain, "mailgun domain")
	flag.StringVar(&smsKey, "sms-key", "80df3a7d-4c8c-ffb4-b197-4dc850443bba", "mailgun domain")
	flag.Parse()
	projectName = *projectNameF
	dbName = projectName
	redisName = projectName
	redisAddr = *redisAddrF
	mongoHost = *mongoHostF
	weedHost = *weedHostF
	weedUrl = fmt.Sprintf("http://%s:%d", weedHost, weedPort)
	salt = *saltF
	a := NewApp()
	defer a.Close()
	a.Run()
}
