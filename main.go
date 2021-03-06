package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime"
	"time"

	"github.com/GeertJohan/go.rice"
	mmodels "github.com/ernado/cymedia/mediad/models"
	"github.com/ernado/cymedia/mediad/query"
	"github.com/ernado/gofbauth"
	"github.com/ernado/gosmsru"
	"github.com/ernado/gotok"
	"github.com/ernado/govkauth"
	"github.com/ernado/poputchiki/activities"
	"github.com/ernado/poputchiki/database"
	"github.com/ernado/poputchiki/models"
	"github.com/ernado/selectel/storage"
	"github.com/ernado/weed"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/rakyll/globalconf"
	"github.com/riobard/go-mailgun"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	salt                           = "salt"
	projectName                    = "poputchiki"
	premiumTime                    = time.Hour * 24 * 30
	vipWeek                        = 400
	vipMonth                       = 1000
	adCost                    uint = 500
	ratingDegradationDuration      = time.Hour * 24 * 2
	ratingUpdateDelta              = time.Second * 3
	statusUpdateTime               = time.Hour * 24
	dbName                         = projectName
	statusesPerDay                 = 1
	statusesPerDayVip              = 3
	startCapital              uint = 0
	AllTemplates              *rice.Box
	dbCity                         = "countries"
	tokenCollection                = "tokens"
	mongoHost                      = "localhost"
	selectelUser                   = "7345"
	selectel                       = false
	selectelKey                    = "TYl2XjMw"
	selectelContainer              = projectName
	robokassaLogin                 = "poputchiki.ru"
	robokassaPassword1             = "pcZKT5Qm84MJAIudLAbR"
	robokassaPassword2             = "8x3cVXUt08Uc9TV70mx3"
	robokassaDescription           = "Пополнение счета Попутчики.ру"
	production                     = false
	processes                      = runtime.NumCPU()
	redisName                      = projectName
	redisAddr                      = ":6379"
	redisQueryKey                  = flag.String("query.key", "poputchiki:conventer:query", "Convertation query key")
	mailKey                        = "key-7520cy18i2ebmrrbs1bz4ivhua-ujtb6"
	mailDomain                     = "mg.cydev.ru"
	etcdHost                       = "http://127.0.0.1:4001"
	smsKey                         = "nil"
	weedHost                       = "127.0.0.1"
	weedPort                       = 9333
	weedUrl                        = fmt.Sprintf("http://%s:%d", weedHost, weedPort)
	OfflineTimeout                 = 20 * time.Minute
	OfflineUpdateTick              = 5 * time.Second
	DublicateUpdatesTimeout        = 5 * time.Minute
	PromoCost                 uint = 50
	mobile                         = flag.Bool("mobile", false, "is mobile api")
	development                    = flag.Bool("dev", false, "is in development")
	sendEmail                      = flag.Bool("email", true, "send registration emails")
	feedbackEmail                  = "info@poputchiki.ru"
	popiki                         = map[int]int{
		100:  30,
		300:  90,
		500:  140,
		1000: 280,
		3000: 800,
	}
)

func getHash(password string, s string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + s))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

type Application struct {
	session      *mgo.Session
	p            *redis.Pool
	m            *martini.ClassicMartini
	db           models.DataBase
	adapter      *weed.Adapter
	updater      models.Updater
	emailUpdater *EmailUpdater
	done         chan bool
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

func NewTestApp() *Application {
	*development = true
	redisName = "poputchiki-test"
	dbName = "poputchiki-test"
	return NewApp()
}

type selectelAdapter struct {
	api storage.ContainerAPI
}

func (s selectelAdapter) GetUrl(name string) (string, error) {
	return s.api.URL(name), nil
}

func (s selectelAdapter) URL(name string) (string, error) {
	return s.GetUrl(name)
}

func (s selectelAdapter) Upload(reader io.Reader, t, format string) (fid string, purl string, size int64, err error) {
	fid = bson.NewObjectId().Hex()
	err = s.api.Upload(reader, fid, t+"/"+format)
	purl = s.api.URL(fid)
	return
}

func GetAdapter(api storage.API) models.StorageAdapter {
	container, err := api.CreateContainer(selectelContainer, false)
	if err != nil {
		log.Fatal(err)
	}
	return selectelAdapter{container}
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

	queryClient, err := query.NewRedisClient(weedUrl, redisAddr, *redisQueryKey, redisQueryRespKey)
	if err != nil {
		log.Fatal(err)
	}

	root := "/api"
	if *mobile {
		root = "/api/mobile"
	}
	activityEngine := activities.New(db, ratingDegradationDuration)
	smsClient := gosmsru.New(smsKey)
	m.Map(smsClient)
	mailgunClient := mailgun.New(mailKey)
	m.Map(mailgunClient)

	templates := rice.MustFindBox("static/html/letters")
	AllTemplates = rice.MustFindBox("static/html")

	m.Map(queryClient)
	m.Map(&gofbauth.Client{"1518821581670594", "97161fd30ed48e5a3e25811ed02d0f3a", "http://poputchiki.ru" + root + "/auth/fb/redirect", "email,user_birthday"})
	m.Map(&govkauth.Client{"4456019", "0F4CUYU2Iq9H7YhANtdf", "http://poputchiki.ru" + root + "/auth/vk/redirect", "offline,email"})
	m.Use(JsonEncoder)
	m.Use(JsonEncoderWrapper)
	m.Use(TokenWrapper)
	m.Use(WebpWrapper)
	m.Use(AudioWrapper)
	m.Use(VideoWrapper)
	m.Use(PaginationWrapper)
	m.Use(ParserWrapper)
	m.Map(tokenStorage)
	weedAdapter := weed.NewAdapter(weedUrl)
	var adapter models.StorageAdapter
	adapter = weedAdapter
	if selectel {
		log.Println("[selectel]", "using selectel adapter")
		selectelApi, err := storage.New(selectelUser, selectelKey)
		if err != nil {
			log.Fatal(err)
		}
		adapter = GetAdapter(selectelApi)
	}
	// m.Map(weedAdapter)
	m.Map(adapter)
	m.Map(realtime)
	m.Use(AutoUpdaterWrapper)
	emailUpdater := &EmailUpdater{db, mailgunClient, templates, weedAdapter}
	updater := &RealtimeUpdater{db, realtime, emailUpdater, &PushNotificationsUpdater{db, weedAdapter}}
	m.MapTo(updater, (*models.Updater)(nil))
	m.Map(db)
	m.Use(activityEngine.Wrapper)
	m.Map(models.GetMailDispatcher(templates, "noreply@"+mailDomain, mailgunClient, db))
	m.Map(NewTransactionHandler(p, session.DB(dbName), robokassaLogin, robokassaPassword1, robokassaPassword2))

	staticOptions := martini.StaticOptions{Prefix: "/api/static/"}
	m.Use(martini.Static("static", staticOptions))
	m.Use(AdminWrapper)
	m.Use(models.ContextWrapper)

	m.Get("/robots.txt", Robots)
	m.Get("/sitemap.xml", Sitemap)

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
	m.Get(root+"/system", GetSystemStatus)
	m.Group(root, func(r martini.Router) {
		r.Get("/cities", GetCities)
		r.Get("/places", GetPlaces)
		r.Get("/citypairs", GetCityPairs)
		r.Get("/countries", GetCountries)
		r.Get("/confirm/email/:token", ConfirmEmail)
	})
	m.Group(root, func(r martini.Router) {
		r.Get("/stripe", GetStripe)
		r.Group("/user/:id", func(r martini.Router) {
			r.Get("", GetUser)
			r.Get("/status", GetCurrentStatus)
			r.Get("/login", NeedAdmin, AdminLogin)
			r.Put("/messages", NeedAuth, SendMessage)
			r.Get("/messages", GetMessagesFromUser)
			r.Delete("/messages", NeedAuth, RemoveChat)
			r.Post("/invite", NeedAuth, SendInvite)
			r.Get("/chats", GetChats)
			r.Get("/photo", GetUserPhoto)
			r.Get("/video", GetUserVideo)
			r.Get("/media", GetUserMedia)
			r.Group("", func(d martini.Router) {
				d.Patch("", UpdateUser)
				d.Put("", UpdateUser)
				d.Post("", UpdateUser)

				d.Post("/fav", AddToFavorites)
				d.Post("/present/:title", SendPresent)
				d.Get("/present", GetUserPresents)
				d.Put("/fav", AddToFavorites)
				d.Delete("/fav", RemoveFromFavorites)
				d.Get("/fav", GetFavorites)

				d.Get("/blacklist", GetBlacklisted)
				d.Post("/blacklist", AddToBlacklist)
				d.Put("/blacklist", AddToBlacklist)
				d.Delete("/blacklist", RemoveFromBlacklist)

				d.Post("/guests", AddToGuests)
				d.Put("/guests", AddToGuests)
				d.Get("/guests", GetGuests)
				d.Get("/unread", GetUnreadCount)
				d.Get("/followers", GetFollowers)

			}, NeedAuth, IdEqualityRequired)

		}, IdWrapper)
	})
	m.Get(root+"/pay/success", RobokassaSuccessHandler)
	m.Group(root, func(r martini.Router) {
		r.Post("/present/add", NeedAdmin, AddPresent)
		r.Delete("/present/:id", NeedAdmin, IdWrapper, RemovePresent)
		r.Post("/present/:id", NeedAdmin, IdWrapper, UpdatePresent)
		r.Get("/present", GetAllPresents)
		r.Get("/admin", NeedAdmin, AdminView)
		r.Get("/admin/photo", NeedAdmin, PhotoView)
		r.Get("/admin/messages", NeedAdmin, AdminMessages)
		r.Get("/admin/presents", NeedAdmin, AdminPresents)
		r.Get("/confirm/phone/start", ConfirmPhoneStart)
		r.Get("/confirm/phone/:token", ConfirmPhone)
		r.Post("/feedback", Feedback)
		r.Post("/travel", WantToTravel)
		r.Post("/pay/:value", GetTransactionUrl)
		r.Get("/pay/:value", GetTransactionUrl)

		r.Post("/push/:system/:token", AddToken)
		r.Delete("/push/:system/:token", RemoveToken)

		r.Get("/topup", TopUp)
		r.Post("/topup", TopUp)

		r.Get("/token", GetToken)
		r.Post("/vip/:duration", EnableVip)

		r.Get("/user", GetCurrentUser)

		r.Get("/chat/:user/:chat", NeedAdmin, GetChat)
		r.Get("/users/:email", NeedAdmin, GetUsersByEmail)

		r.Post("/stripe", AddStripeItem)
		r.Put("/stripe", AddStripeItem)

		r.Get("/ads", PaginationWrapper, AdvGet)
		r.Post("/ads", AdvAdd)
		r.Delete("/ads/:id", IdWrapper, AdvRemove)

		r.Get("/updates/counters", GetCounters)
		r.Get("/updates", GetUpdates)
		r.Post("/updates", SetUpdatesRead)
		r.Delete("/updates", SetUpdatesRead)
		r.Post("/updates/:id", IdWrapper, SetUpdateRead)

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
		r.Delete("/video/:id", IdWrapper, RemoveVideo)
		r.Delete("/audio/:id", IdWrapper, RemoveAudio)
		r.Get("/video/:id", IdWrapper, GetVideo)
		r.Get("/photo/:id", IdWrapper, GetPhoto)
		r.Post("/photo", UploadPhoto)
		r.Post("/photo-hidden", UploadPhotoHidden)
		r.Get("/realtime", realtime.RealtimeHandler)
		r.Get("/search", SearchPeople)
		r.Get("/photo", SearchPhoto)
		r.Get("/photo-all", AllPhoto)
		r.Post("/photo/:id/like", IdWrapper, LikePhoto)
		r.Get("/photo/:id/like", IdWrapper, GetLikersPhoto)
		r.Delete("/photo/:id/like", IdWrapper, RestoreLikePhoto)
		r.Delete("/photo/:id", IdWrapper, RemovePhoto)
	}, NeedAuth, SetOnlineWrapper)

	a := &Application{session, p, m, db, weedAdapter, updater, emailUpdater, make(chan bool)}
	a.InitDatabase()
	return a
}

func (a *Application) Close() {
	a.p.Close()
}

func (a *Application) newCycle(name string, d time.Duration, callback func(chan bool)) {
	log.Printf("[cycle] starting %s with duration %v\n", name, d)
	defer log.Printf("[cycle] %s stopped\n", name)
	ticker := time.NewTicker(d)
	stop := make(chan bool)
	defer ticker.Stop()
	defer func() {
		r := recover()
		if r != nil {
			log.Printf("[cycle] %s %s\n", name, r)
		}
	}()
	for {
		select {
		case <-ticker.C:
			callback(stop)
		case <-stop:
			log.Printf("[cycle] %s got local stop signal\n", name)
			return
		case <-a.done:
			log.Printf("[cycle] %s got global done signal\n", name)
			return
		}
	}
}

func (a *Application) StatusCycle() {
	callback := func(_ chan bool) {
		i, err := a.db.UpdateAllStatuses()
		if err != nil {
			log.Println("[status]", "status update error", err)
			time.Sleep(time.Second * 5)
		} else {
			if i.Updated != 0 {
				log.Println("[status]", "statuses updated: ", i.Updated)
			}
		}
	}
	a.newCycle("status", OfflineUpdateTick, callback)
}

func (a *Application) VipCycle() {
	callback := func(_ chan bool) {
		i, err := a.db.UpdateAllVip()
		if err != nil {
			log.Println("[updater]", "vip update error", err)
			time.Sleep(time.Second * 5)
		} else {
			if i.Updated != 0 {
				log.Println("[updater]", "vip updated: ", i.Updated)
			}
		}
	}
	a.newCycle("vip updater", OfflineUpdateTick, callback)
}

func (a *Application) NormalizeRatingCycle() {
	a.newCycle("rating", OfflineUpdateTick, func(_ chan bool) {
		i, err := a.db.NormalizeRating()
		if err != nil {
			log.Println("[rating]", "error", err)
			return
		}
		if i.Updated != 0 {
			log.Println("[rating]", "normalized: ", i.Updated)
		}
	})
}

func (a *Application) RatingDegradatingCycle() {
	fullRating := 100.0
	deltaTime := float64(ratingUpdateDelta.Nanoseconds())
	fullTime := float64(ratingDegradationDuration.Nanoseconds())
	rate := fullRating * deltaTime / fullTime
	lastLog := time.Now()
	logRate := time.Second * 30
	a.newCycle("rating", ratingUpdateDelta, func(_ chan bool) {
		start := time.Now()
		i, err := a.db.DegradeRating(rate)
		duration := time.Now().Sub(start)
		if err != nil {
			log.Println("[rating]", "error", err)
		} else {
			if i.Updated != 0 && start.Sub(lastLog) > logRate {
				log.Println("[rating]", "updated: ", i.Updated, "for", duration)
				lastLog = start
			}
		}
	})
}

var redisQueryRespKey = fmt.Sprintf("%s:conventer:resp", projectName)

func (a *Application) PromoCycle() {
	client := &RandomCycle{a.p, a.db}
	client.Cycle()
}

func (a *Application) processConvertResult(resp mmodels.Responce) (err error) {
	db := a.db
	id := bson.ObjectIdHex(resp.Id)
	fid := resp.File
	defer func() {
		log.Println("[dedicated server] processed")
		r := recover()
		if r != nil {
			log.Println("[dedicated server]", r)
		}
	}()
	if resp.Type == "audio" {
		audio := db.GetAudio(id)
		if audio == nil {
			return errors.New("audio not found")
		}
		if resp.Format == "ogg" {
			err = db.UpdateAudioOGG(id, fid)
			audio.AudioOgg = fid
		}
		if resp.Format == "mp3" {
			err = db.UpdateAudioAAC(id, fid)
			audio.AudioAac = fid
		}
		if !resp.Success || (len(audio.AudioAac) > 0 && len(audio.AudioOgg) > 0) {
			log.Printf("Sending audio %+v", audio)
			u := models.NewUpdate(audio.User, audio.User, "audio", audio)
			if err := a.updater.Push(u); err != nil {
				return err
			}
		}
		if !resp.Success {
			db.RemoveAudio(id)
		}
	}
	if resp.Type == "video" {
		video := db.GetVideo(id)
		if video == nil {
			return errors.New("video not found")
		}
		if resp.Format == "webm" {
			err = db.UpdateVideoWebm(id, fid)
			video.VideoWebm = fid
		}
		if resp.Format == "mp4" {
			err = db.UpdateVideoMpeg(id, fid)
			video.VideoMpeg = fid
		}
		if !resp.Success || (len(video.VideoWebm) > 0 && len(video.VideoMpeg) > 0) {
			log.Printf("Sending video %+v", video)
			u := models.NewUpdate(video.User, video.User, "video", video)
			if err := a.updater.Push(u); err != nil {
				return err
			}
		}
		if !resp.Success {
			db.RemoveVideo(id, video.User)
			return nil
		}
	}
	if resp.Type == "thumbnail" {
		thumbnail, err := ExportThumbnail(a.adapter, resp.File)
		if err != nil {
			log.Println(err)
		} else {
			err = db.UpdateVideoThumbnail(id, thumbnail)
		}
	}
	return err
}

func (a *Application) ConvertResultListener() {
	log.Println("[conventer] started conventer", *redisQueryKey, "->", redisQueryRespKey)
	q, err := query.NewRedisResponceQuery(redisAddr, redisQueryRespKey)
	if err != nil {
		log.Println("[conventer] unable to create result listener", err)
		return
	}
	results := make(chan mmodels.Responce)
	go func() {
		for {
			r, err := q.Pull()
			if err != nil {
				log.Println("[conventer]", err)
				return
			}
			if results == nil {
				return
			}
			results <- r
		}
	}()
	for {
		select {
		case r := <-results:
			if err := a.processConvertResult(r); err != nil {
				log.Println("[conventer]", err)
			}
		case <-a.done:
			log.Println("[conventer]", "done")
			close(results)
			return
		}
	}
	return
}

func (a *Application) Run() {
	go a.StatusCycle()
	go a.VipCycle()
	go a.ConvertResultListener()
	go a.RatingDegradatingCycle()
	go a.NormalizeRatingCycle()
	// go a.PromoCycle()
	a.m.Run()
}

func (a *Application) DropDatabase() {
	a.db.Drop()
	a.InitDatabase()
}

func (a *Application) InitDatabase() {
	a.db.Init()
}

func (a *Application) Reset() {
	a.DropDatabase()
	a.InitDatabase()
}

func (a *Application) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	a.m.ServeHTTP(res, req)
}

func (a *Application) Serve(req *http.Request) http.ResponseWriter {
	res := httptest.NewRecorder()
	a.m.ServeHTTP(res, req)
	return res
}

func (a *Application) ServeJSON(req *http.Request, value interface{}) error {
	res := httptest.NewRecorder()
	a.m.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		return errors.New(fmt.Sprintf("Bad code %d", res.Code))
	}
	decoder := json.NewDecoder(res.Body)
	return decoder.Decode(value)
}

func (a *Application) SendJSON(method, url string, input, output interface{}) error {
	res := httptest.NewRecorder()
	var body io.Reader = nil
	if input != nil {
		j, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, url, body)
	if output != nil {
		req.Header.Add("Accept", JSON_HEADER)
	}
	if input != nil {
		req.Header.Add("Content-type", JSON_HEADER)
	}
	if err != nil {
		return err
	}
	a.m.ServeHTTP(res, req)
	decoder := json.NewDecoder(res.Body)
	if res.Code != http.StatusOK {
		result := new(models.Error)
		result.Code = 500
		result.Text = "panic"
		decoder.Decode(result)
		return result
	}
	if output != nil {
		return decoder.Decode(output)
	}
	return nil
}

func (a *Application) Process(token *gotok.Token, method, url string, input, output interface{}) error {
	res := httptest.NewRecorder()
	var body io.Reader
	if input != nil {
		j, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	if output != nil {
		req.Header.Add("Accept", JSON_HEADER)
	}
	if input != nil {
		req.Header.Add("Content-type", JSON_HEADER)
	}
	if token != nil {
		req.AddCookie(token.GetCookie())
	}
	a.m.ServeHTTP(res, req)
	decoder := json.NewDecoder(res.Body)
	if res.Code != http.StatusOK {
		log.Println("http", res.Code)
		result := new(models.Error)
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		result.Code = res.Code
		result.Text = string(b)
		json.Unmarshal(b, result)
		return result
	}
	if output != nil {
		return decoder.Decode(output)
	}
	return nil
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
	flag.BoolVar(&selectel, "selectel", false, "use selectel storage")
	flag.StringVar(&smsKey, "sms-key", "80df3a7d-4c8c-ffb4-b197-4dc850443bba", "sms key")
	flag.StringVar(&etcdHost, "etcd", etcdHost, "etcd host")
	flag.StringVar(&selectelKey, "selectel.key", selectelKey, "Selectel key")
	flag.StringVar(&selectelUser, "selectel.user", selectelUser, "Selectel user")
	// flag.Parse()
	conf, err := globalconf.New("poputchiki")
	if err != nil {
		log.Fatal(err)
	}
	conf.ParseAll()
	projectName = *projectNameF
	dbName = projectName
	redisName = projectName
	redisAddr = *redisAddrF
	mongoHost = *mongoHostF
	weedHost = *weedHostF
	weedUrl = fmt.Sprintf("http://%s:%d", weedHost, weedPort)
	salt = *saltF
	log.Println("[project]", "starting", projectName)
	redisQueryRespKey = fmt.Sprintf("%s:conventer:resp", projectName)
	log.Println("[project] key", redisQueryRespKey)
	a := NewApp()
	defer a.Close()
	a.Run()
}
