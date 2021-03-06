package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/ernado/gotok"
	"github.com/go-martini/martini"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// webp accept flag
type WebpAccept bool

type IsAdmin bool

// video format accept
type VideoAccept string

// audio format accept
type AudioAccept string

type Storage interface {
	URL(string) (string, error)
}

type StorageAdapter interface {
	Storage
	GetUrl(fid string) (url string, err error)
	Upload(reader io.Reader, t, format string) (fid string, purl string, size int64, err error)
}

type Context struct {
	Storage  StorageAdapter
	Token    *gotok.Token
	Video    VideoAccept
	Audio    AudioAccept
	WebP     WebpAccept
	IsAdmin  IsAdmin
	DB       DataBase
	Request  *http.Request
	User     *User
	Renderer Renderer
	Parser   Parser
}

type ContextRenderer struct {
	context Context
}

func addStatus(value interface{}, status int) []byte {
	j, err := json.Marshal(Response{status, value})
	if err != nil {
		log.Println(string(j), err)
		panic(err)
	}
	return j
}
func (context Context) Render(value interface{}) (int, []byte) {
	return context.Renderer.Render(value)
}

func (context Context) Parse(v interface{}) error {
	return context.Parser.Parse(v)
}

type Preparable interface {
	Prepare(context Context) error
}

func (renderer ContextRenderer) Render(value interface{}) (int, []byte) {
	var mobile, ok bool

	start := time.Now()
	defer func() {
		log.Printf("[render] %T mobile=%v preparable=%v | %v\n", value, mobile, ok, time.Now().Sub(start))
	}()

	if strings.Contains(renderer.context.Request.URL.Path, "/api/mobile") {
		mobile = true
	}

	preparable, ok := value.(Preparable)

	if ok {
		prepareStart := time.Now()
		if err := preparable.Prepare(renderer.context); err != nil {
			log.Println("[prepare]", err)
		}
		log.Println("[render] [prepare]", time.Now().Sub(prepareStart))
	}

	j, err := json.Marshal(value)
	if err != nil {
		j, err = json.Marshal(ErrorMarshal)
		if err != nil {
			log.Println(err)
			panic(err)
		}
		return ErrorMarshal.Code, j
	}
	switch v := value.(type) {
	case Error:
		if v.Code == http.StatusInternalServerError {
			log.Println(v)
			debug.PrintStack()
		}
		if mobile {
			return http.StatusOK, addStatus(value, v.Code)
		}
		return v.Code, j
	default:
		if mobile {
			return http.StatusOK, addStatus(value, 0)
		}
		return http.StatusOK, j
	}
}

type Renderer interface {
	Render(value interface{}) (int, []byte)
}

func ContextWrapper(context martini.Context, parser Parser, t *gotok.Token, webp WebpAccept, video VideoAccept, audio AudioAccept, admin IsAdmin, db DataBase, request *http.Request, storage StorageAdapter) {
	c := Context{}
	c.Storage = storage
	c.Request = request
	c.IsAdmin = admin
	c.Token = t
	c.Audio = audio
	c.Video = video
	c.DB = db
	c.WebP = webp
	c.Parser = parser
	if t != nil {
		c.User = c.DB.Get(t.Id)
	}
	var renderer Renderer
	renderer = ContextRenderer{c}
	c.Renderer = renderer
	context.Map(c)
	context.Map(renderer)
}

var (
	VaWebm VideoAccept = "webm"
	VaMp4  VideoAccept = "mp4"
	AaAac  AudioAccept = "acc"
	AaOgg  AudioAccept = "ogg"
)

type SearchResult struct {
	Result interface{} `json:"result"`
	Count  int         `json:"count"`
}

type DataBase interface {
	// GetAll() []*User
	RandomUser() (*User, error)
	AllUsers() []*User
	GetUsername(username string) *User
	Get(id bson.ObjectId) *User
	Add(u *User) error
	GetUsersByEmail(email string) (Users, error)
	Update(id bson.ObjectId, update bson.M) (*User, error)
	AvatarRemove(user, id bson.ObjectId) error
	// Delete(id bson.ObjectId) error
	AddToFavorites(id bson.ObjectId, favId bson.ObjectId) error
	RemoveFromFavorites(id bson.ObjectId, favId bson.ObjectId) error
	GetFavorites(id bson.ObjectId) []*User
	GetAllUsersWithFavorite(id bson.ObjectId) ([]*User, error)

	AddGuest(id bson.ObjectId, guest bson.ObjectId) error
	// GetAllGuests(id bson.ObjectId) ([]*User, error)
	GetAllGuestUsers(id bson.ObjectId) ([]*GuestUser, error)

	AddMessage(m *Message) error
	AddInvite(i *Invite) error
	GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId, paginaton Pagination) (messages Messages, err error)
	GetMessage(id bson.ObjectId) (message *Message, err error)
	RemoveMessage(id bson.ObjectId) error
	GetChats(id bson.ObjectId) ([]*Dialog, error)
	SetRead(user, id bson.ObjectId) error
	SetReadMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) error
	GetUnreadCount(id bson.ObjectId) (int, error)
	RemoveChat(userReciever bson.ObjectId, userOrigin bson.ObjectId) error

	AddToBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error
	RemoveFromBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error
	GetBlacklisted(id bson.ObjectId) []*User

	IncBalance(id bson.ObjectId, amount uint) error
	DecBalance(id bson.ObjectId, amount uint) error

	SetVipTill(id bson.ObjectId, t time.Time) error
	SetVip(id bson.ObjectId, vip bool) error

	Online() int
	RegisteredCount(duration time.Duration) int
	SetOnline(id bson.ObjectId) error
	SetOffline(id bson.ObjectId) error

	SetRating(id bson.ObjectId, rating float64) error
	ChangeRating(id bson.ObjectId, delta float64) error

	// statuses
	// api/user/:id/status/
	GetCurrentStatus(user bson.ObjectId) (status *Status, err error)
	// api/status/:id/
	GetStatus(id bson.ObjectId) (status *Status, err error)
	GetLastStatuses(count int) (status []*Status, err error)
	AddStatus(u bson.ObjectId, text string) (*Status, error)
	UpdateStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) (*Status, error)
	RemoveStatusSecure(user bson.ObjectId, id bson.ObjectId) error
	AddLikeStatus(user bson.ObjectId, target bson.ObjectId) error
	RemoveLikeStatus(user bson.ObjectId, target bson.ObjectId) error
	GetLikesStatus(id bson.ObjectId) []*User
	GetLastDayStatusesAmount(id bson.ObjectId) (int, error)
	// api/status/:id/comment/:id

	AddPhoto(user bson.ObjectId, image, thumbnail string) (*Photo, error)
	AddPhotoHidden(user bson.ObjectId, image, thumbnail string) (*Photo, error)

	// api/photo/:id
	GetPhoto(photo bson.ObjectId) (*Photo, error)
	RemovePhoto(user bson.ObjectId, id bson.ObjectId) error
	SearchPhoto(q *SearchQuery, pagination Pagination) ([]*Photo, error)
	SearchAllPhoto(pagination Pagination) ([]*Photo, int, error)
	AddLikePhoto(user bson.ObjectId, target bson.ObjectId) error
	RemoveLikePhoto(user bson.ObjectId, target bson.ObjectId) error
	GetLikesPhoto(id bson.ObjectId) []*User
	GetUserPhoto(user bson.ObjectId) ([]*Photo, error)

	AddFile(file *File) (*File, error)

	AddVideo(video *Video) (*Video, error)
	GetVideo(id bson.ObjectId) *Video
	GetUserVideo(id bson.ObjectId) ([]*Video, error)
	AddLikeVideo(user bson.ObjectId, target bson.ObjectId) error
	RemoveLikeVideo(user bson.ObjectId, target bson.ObjectId) error
	GetLikesVideo(id bson.ObjectId) []*User
	UpdateVideoWebm(id bson.ObjectId, fid string) error
	UpdateVideoMpeg(id bson.ObjectId, fid string) error
	UpdateVideoThumbnail(id bson.ObjectId, thumbnail string) error
	RemoveVideo(user bson.ObjectId, id bson.ObjectId) error

	AddAudio(audio *Audio) (*Audio, error)
	GetAudio(id bson.ObjectId) *Audio
	UpdateAudioAAC(id bson.ObjectId, fid string) error
	UpdateAudioOGG(id bson.ObjectId, fid string) error
	RemoveAudio(id bson.ObjectId) error
	RemoveAudioSecure(user, id bson.ObjectId) error

	AddStripeItem(i *StripeItem, media interface{}) (*StripeItem, error)
	GetStripeItem(id bson.ObjectId) (*StripeItem, error)
	GetStripe(count, offset int) ([]*StripeItem, error)

	Search(q *SearchQuery, pagination Pagination) ([]*User, int, error)
	SearchStatuses(q *SearchQuery, pagination Pagination) ([]*Status, error)

	NewConfirmationToken(id bson.ObjectId) *EmailConfirmationToken
	GetConfirmationToken(token string) *EmailConfirmationToken
	NewConfirmationTokenValue(id bson.ObjectId, token string) *EmailConfirmationToken
	ConfirmEmail(id bson.ObjectId) error
	ConfirmPhone(id bson.ObjectId) error

	UpdateAllStatuses() (*mgo.ChangeInfo, error)
	SetLastActionNow(id bson.ObjectId) error

	CountryExists(name string) bool
	CityExists(name string) bool
	GetCities(start, country string) (cities []string, err error)
	GetCountries(start string) (countries []string, err error)
	GetPlaces(start string) (places []string, err error)
	GetCityPairs(start string) (cities Cities, err error)

	Init()
	Drop()
	Salt() string

	SetAvatar(user, avatar bson.ObjectId) error

	UpdateAllVip() (*mgo.ChangeInfo, error)
	DegradeRating(amount float64) (*mgo.ChangeInfo, error)
	NormalizeRating() (*mgo.ChangeInfo, error)
	GetActivityCount(user bson.ObjectId, key string, duration time.Duration) (count int, err error)
	AddActivity(user bson.ObjectId, key string) error
	UserIsSubscribed(id bson.ObjectId, subscription string) (bool, error)
	AddUpdateDirect(u *Update) (*Update, error)
	GetUpdatesCount(destination bson.ObjectId) ([]*UpdateCounter, error)
	GetUpdates(destination bson.ObjectId, t string, pagination Pagination) ([]*Update, error)
	SetUpdateRead(destination, id bson.ObjectId) error
	SetUpdatesRead(destination bson.ObjectId, t string) error
	IsUpdateDublicate(origin, destination bson.ObjectId, t string, duration time.Duration) (bool, error)
	GetLastMessageIdFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) (id bson.ObjectId, err error)

	RemoveAndroidToken(id bson.ObjectId, token string) error
	RemoveIosToken(id bson.ObjectId, token string) error
	AddAndroidToken(id bson.ObjectId, token string) error
	AddIosToken(id bson.ObjectId, token string) error

	AddPresent(present *Present) error
	RemovePresent(id bson.ObjectId) error
	GetPresentByType(t string) (*Present, error)
	GetAllPresents() ([]*Present, error)
	SendPresent(origin, destination bson.ObjectId, t string) (*PresentEvent, error)
	GetUserPresents(destination bson.ObjectId) ([]*PresentEvent, error)
	GetPresent(id bson.ObjectId) (*Present, error)
	UpdatePresent(id bson.ObjectId, present *Present) (*Present, error)

	GetAds(count, offset int) (Stripe, error)
	GetAdvertisment(id bson.ObjectId) (*StripeItem, error)
	AddAdvertisement(user bson.ObjectId, i *StripeItem, media interface{}) (*StripeItem, error)
	RemoveAdvertisment(user, id bson.ObjectId) error

	ActiveCount(duration time.Duration) int

	// all
	GetAllAudio() ([]*Audio, error)
	GetAllVideo() ([]*Video, error)
}

type RealtimeInterface interface {
	Updater
	RealtimeHandler(w http.ResponseWriter, context Context) (int, []byte)
	// PushAll(update Update) error
}

type Updater interface {
	Push(update Update) error
}

type AutoUpdater interface {
	Push(destination bson.ObjectId, body interface{}) error
}

type PrepareInterface interface {
	Prepare(context Context) error
	Url() string
}

type PhotoSlice []*Photo
type VideoSlice []*Video

func (m PhotoSlice) Prepare(context Context) error {
	var e error
	for _, elem := range m {
		if err := elem.Prepare(context); err != nil {
			log.Println(err)
			e = err
		}
	}
	return e
}

func (m VideoSlice) Prepare(context Context) error {
	var e error
	for _, elem := range m {
		if err := elem.Prepare(context); err != nil {
			log.Println(err)
			e = err
		}
	}
	return e
}

type MailSender interface {
	Send()
}

func Random(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type Media struct {
	Media interface{} `json:"media"`
	Time  time.Time   `json:"time"`
	Type  string      `json:"type"`
}

type MediaSlice []Media

func (a MediaSlice) Len() int {
	return len(a)
}

func (a MediaSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a MediaSlice) Less(i, j int) bool {
	return a[j].Time.Before(a[i].Time)
}

func MakeMediaSlice(photo []*Photo, video []*Video) (media MediaSlice) {
	for _, p := range photo {
		media = append(media, Media{p, p.Time, "photo"})
	}
	for _, v := range video {
		media = append(media, Media{v, v.Time, "video"})
	}
	sort.Sort(media)
	return
}
