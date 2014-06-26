package main

import (
	"encoding/json"
	"errors"
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	SEASON_SUMMER = "summer"
	SEASON_WINTER = "winter"
	SEASON_AUTUMN = "autumn"
	SEASON_SPRING = "spring"
	SEX_MALE      = "male"
	SEX_FEMALE    = "female"
	AGE_MAX       = 100
	AGE_MIN       = 18
	GROWTH_MAX    = 300
	WEIGHT_MAX    = 1000
)

var UserWritableFields = []string{"name", "email", "phone", "avatar", "birthday", "seasons",
	"city", "country", "weight", "growth", "destinations", "sex"}

type User struct {
	Id             bson.ObjectId   `json:"id"                     bson:"_id"`
	Name           string          `json:"name,omitempty"         bson:"name,omitempty"`
	Sex            string          `json:"sex,omitempty"          bson:"sex,omitempty"`
	Email          string          `json:"email,omitempty"        bson:"email,omitempty"`
	EmailConfirmed bool            `json:"email_confirmed"        bson:"email_confirmed"`
	Phone          string          `json:"phone,omitempty"        bson:"phone,omitempty"`
	PhoneConfirmed bool            `json:"phone_confirmed"        bson:"phone_confirmed"`
	IsSponsor      bool            `json:"is_sponsor"             bson:"is_sponsor"`
	IsHost         bool            `json:"is_host"                bson:"is_host"`
	Password       string          `json:"-"                      bson:"password"`
	Online         bool            `json:"online,omitempty"       bson:"online,omitempty"`
	AvatarUrl      string          `json:"avatar_url,omitempty"   bson:"-"`
	Avatar         bson.ObjectId   `json:"avatar,omitempty"       bson:"avatar,omitempty"`
	AvatarWebp     string          `json:"-"                      bson:"image_webp,omitempty"`
	AvatarJpeg     string          `json:"-"                      bson:"image_jpeg,omitempty"`
	Balance        uint            `json:"balance,omitempty"      bson:"balance,omitempty"`
	Age            int             `json:"age,omitempty"          bson:"-"`
	Birthday       time.Time       `json:"birthday,omitempty"     bson:"birthday,omitempty"`
	City           string          `json:"city,omitempty"         bson:"city,omitempty"`
	Country        string          `json:"country,omitempty"      bson:"country,omitempty"`
	Weight         uint            `json:"weight,omitempty"       bson:"weight,omitempty"`
	Growth         uint            `json:"growth,omitempty"       bson:"growth,omitempty"`
	Destinations   []string        `json:"destinations,omitempty" bson:"destinations,omitempty"`
	Seasons        []string        `json:"seasons,omitempty"      bson:"seasons,omitempty"`
	LastAction     time.Time       `json:"last_action,omitempty"  bson:"lastaction,omitempty"`
	StatusUpdate   time.Time       `json:"-"                      bson:"statusupdate,omitempty"`
	Favorites      []bson.ObjectId `json:"favorites,omitempty"    bson:"favorites,omitempty"`
	Blacklist      []bson.ObjectId `json:"blacklist,omitempty"    bson:"blacklist,omitempty"`
	Countries      []string        `json:"countries,omitempty"    bson:"countries,omitempty"`
}

// additional user info
type UserInfo struct {
	Id    bson.ObjectId `json:"id"     bson:"_id"`
	About string        `json:"about"  bson:"about"`
}

type SearchQuery struct {
	Sex          string
	Seasons      []string
	Destinations []string
	AgeMin       int
	AgeMax       int
	WeightMin    int
	WeightMax    int
	GrowthMin    int
	GrowthMax    int
	City         string
	Country      string
	Text         string
}

func NewQuery(q url.Values) (*SearchQuery, error) {
	nQ := make(map[string]interface{})
	for key, value := range q {
		if len(value) == 1 {
			v := value[0]
			vInt, err := strconv.Atoi(v)
			if err != nil {
				nQ[key] = v
			} else {
				nQ[key] = vInt
			}
		} else {
			nQ[key] = value
		}
	}
	j, err := json.Marshal(nQ)
	if err != nil {
		return nil, err
	}
	query := &SearchQuery{}
	err = json.Unmarshal(j, query)
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (q *SearchQuery) ToBson() bson.M {
	query := []bson.M{}
	if q.Sex != BLANK {
		query = append(query, bson.M{"sex": q.Sex})
	}
	if len(q.Seasons) > 0 {
		query = append(query, bson.M{"seasons": bson.M{"$in": q.Seasons}})
	}

	if len(q.Destinations) > 0 {
		query = append(query, bson.M{"destination": bson.M{"$in": q.Destinations}})
	}

	if q.AgeMax == 0 {
		q.AgeMax = AGE_MAX
	}

	if q.AgeMin == 0 {
		q.AgeMin = AGE_MIN
	}

	if q.AgeMin != AGE_MIN || q.AgeMax != AGE_MAX {
		now := time.Now()
		tMax := now.AddDate(-q.AgeMax, 0, 0)
		tMin := now.AddDate(-q.AgeMin, 0, 0)
		query = append(query, bson.M{"birthday": bson.M{"$gte": tMax, "$lte": tMin}})
	}

	if q.GrowthMax == 0 {
		q.GrowthMax = GROWTH_MAX
	}

	if q.GrowthMax != GROWTH_MAX || q.GrowthMin != 0 {
		query = append(query, bson.M{"growth": bson.M{"$gte": q.GrowthMin, "$lte": q.GrowthMax}})
	}

	if q.WeightMax == 0 {
		q.WeightMax = WEIGHT_MAX
	}

	if q.WeightMax != WEIGHT_MAX || q.WeightMin != 0 {
		query = append(query, bson.M{"weight": bson.M{"$gte": q.WeightMin, "$lte": q.WeightMax}})
	}

	if q.City != BLANK {
		query = append(query, bson.M{"city": q.City})
	}

	if q.Country != BLANK && q.City == BLANK {
		query = append(query, bson.M{"country": q.Country})
	}

	fullQuery := bson.M{"$and": query}
	m, _ := json.Marshal(fullQuery)
	log.Println(string(m))
	return fullQuery
}

type Pagination struct {
	Count  int
	Offset int
}

const (
	BLANK = ""
)

func UserFromForm(r *http.Request) *User {
	u := User{}
	u.Id = bson.NewObjectId()
	u.Email = r.FormValue(FORM_EMAIL)
	u.Password = getHash(r.FormValue(FORM_PASSWORD))
	u.Phone = r.FormValue(FORM_PHONE)
	u.Name = r.FormValue(FORM_FIRSTNAME)
	return &u
}

func UpdateUserFromForm(r *http.Request, u *User) {
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(u)
}

func (u *User) CleanPrivate() {
	u.Password = ""
	u.Phone = ""
	u.Email = ""
	u.Favorites = nil
	u.Blacklist = nil
	u.Balance = 0
}

func diff(t1, t2 time.Time) (years int) {
	t2 = t2.AddDate(0, 0, 1) // advance t2 to make the range inclusive

	for t1.AddDate(years, 0, 0).Before(t2) {
		years++
	}
	years--
	return
}

func (u *User) SetAvatarUrl(adapter *weed.Adapter, db UserDB, webp WebpAccept) {
	photo, err := db.GetPhoto(u.Avatar)
	if err == nil {
		suffix := ".jpg"
		fid := photo.ThumbnailJpeg
		if webp {
			fid = photo.ThumbnailWebp
			suffix = ".webp"
		}
		url, _ := adapter.GetUrl(fid)
		u.AvatarUrl = url + suffix
	}
}

func (u *User) Prepare(adapter *weed.Adapter, db UserDB, webp WebpAccept) {
	u.SetAvatarUrl(adapter, db, webp)
	now := time.Now()
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	if u.Birthday.Unix() != 0 {
		u.Age = diff(u.Birthday, now)
	}
}

func (p Photo) Prepare(adapter *weed.Adapter, webp WebpAccept, _ VideoAccept, _ AudioAccept) error {
	var err error
	if webp {
		p.ThumbnailUrl, err = adapter.GetUrl(p.ThumbnailWebp)
		if err != nil {
			return err
		}
		p.ImageUrl, err = adapter.GetUrl(p.ImageWebp)
	} else {
		p.ThumbnailUrl, err = adapter.GetUrl(p.ThumbnailJpeg)
		if err != nil {
			return err
		}
		p.ImageUrl, err = adapter.GetUrl(p.ImageJpeg)
	}
	return err
}

func (v Video) Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, _ AudioAccept) error {
	var err error
	if video == VA_WEBM {
		v.VideoUrl, err = adapter.GetUrl(v.VideoWebm)
	} else if video == VA_MP4 {
		v.VideoUrl, err = adapter.GetUrl(v.VideoMpeg)
	}
	if err != nil {
		return err
	}
	if webp {
		v.ThumbnailUrl, err = adapter.GetUrl(v.ThumbnailWebp)
	} else {
		v.ThumbnailUrl, err = adapter.GetUrl(v.ThumbnailJpeg)
	}
	return err
}

func (audio Audio) Prepare(adapter *weed.Adapter, _ WebpAccept, _ VideoAccept, a AudioAccept) error {
	var err error
	if a == AA_ACC {
		audio.AudioUrl, err = adapter.GetUrl(audio.AudioAac)
	} else if a == AA_OGG {
		audio.AudioUrl, err = adapter.GetUrl(audio.AudioOgg)
	}
	return err
}

func (u StatusUpdate) Prepare(adapter *weed.Adapter, webp WebpAccept) (err error) {
	if webp {
		u.ImageUrl, err = adapter.GetUrl(u.ImageWebp)
	} else {
		u.ImageUrl, err = adapter.GetUrl(u.ImageJpeg)
	}
	return err
}

func (stripe *StripeItem) Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, audio AudioAccept) error {
	var err error
	if webp {
		stripe.ImageUrl, err = adapter.GetUrl(stripe.ImageWebp)
	} else {
		stripe.ImageUrl, err = adapter.GetUrl(stripe.ImageJpeg)
	}
	if err != nil {
		return err
	}

	var media PrepareInterface
	switch stripe.Type {
	case "video":
		media = stripe.Media.(Video)
	case "audio":
		media = stripe.Media.(Audio)
	case "photo":
		media = stripe.Media.(Photo)
	default:
		return errors.New("bad type")
	}

	return media.Prepare(adapter, webp, video, audio)
}

type Guest struct {
	Id    bson.ObjectId `json:"id"    bson:"_id"`
	User  bson.ObjectId `json:"user"  bson:"user"`
	Guest bson.ObjectId `json:"guest" bson:"guest"`
	Time  time.Time     `json:"time"  bson:"time"`
}

type Message struct {
	Id          bson.ObjectId `json:"id"          bson:"_id"`
	User        bson.ObjectId `json:"user"        bson:"user"`
	Origin      bson.ObjectId `json:"origin"      bson:"origin"`
	Destination bson.ObjectId `json:"destination" bson:"destination"`
	Time        time.Time     `json:"time"        bson:"time"`
	Text        string        `json:"text"        bson:"text"`
}

type RealtimeEvent struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
	Time time.Time   `json:"time"`
}

type ProgressMessage struct {
	Id       bson.ObjectId `json:"id,omitempty"`
	Progress float32       `json:"progress"`
}

type MessageSendBlacklisted struct {
	Id bson.ObjectId `json:"id"`
}
type Comment struct {
	Id   bson.ObjectId `json:"id"   bson:"_id"`
	User bson.ObjectId `json:"user" bson:"user"`
	Text string        `json:"text" bson:"text"`
	Time time.Time     `json:"time" bson:"time"`
}

type LoginCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type StatusUpdate struct {
	Id        bson.ObjectId `json:"id"            bson:"_id"`
	User      bson.ObjectId `json:"user"          bson:"user"`
	Name      string        `json:"name"          bson:"-"`
	Age       int           `json:"age,omitempty" bson:"-"`
	Time      time.Time     `json:"time"          bson:"time"`
	Text      string        `json:"text"          bson:"text"`
	ImageWebp string        `json:"-"             bson:"image_webp"`
	ImageJpeg string        `json:"-"             bson:"image_jpeg"`
	ImageUrl  string        `json:"url"           bson:"-"`
	Comments  []Comment     `json:"comments"      bson:"comments"`
}

type StripeItem struct {
	Id        bson.ObjectId `json:"id"                  bson:"_id"`
	User      bson.ObjectId `json:"user"                bson:"user"`
	Name      string        `json:"name"                bson:"name"`
	Age       int           `json:"age,omitempty"       bson:"-"`
	ImageWebp string        `json:"-"                   bson:"image_webp"`
	ImageJpeg string        `json:"-"                   bson:"image_jpeg"`
	ImageUrl  string        `json:"image_url,omitempty" bson:"-"`
	Type      string        `json:"type"                bson:"type"`
	Url       string        `json:"url"                 bson:"-"`
	Media     interface{}   `json:"-"                   bson:"media"`
	Countries []string      `json:"countries,omitempty" bson:"countries,omitempty"`
	Time      time.Time     `json:"time"                bson:"time"`
}

type File struct {
	Id   bson.ObjectId `json:"id,omitempty"  bson:"_id,omitempty"`
	Fid  string        `json:"fid"           bson:"fid"`
	User bson.ObjectId `json:"user"          bson:"user"`
	Time time.Time     `json:"time"          bson:"time"`
	Type string        `json:"type"          bson:"type"`
	Size int64         `json:"size"          bson:"size"`
	Url  string        `json:"url,omitempty" bson:"-"`
}

type Image struct {
	Id  bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Fid string        `json:"fid"          bson:"fid"`
	Url string        `json:"url"          bson:"url,omitempty"`
}

type Album struct {
	Id        bson.ObjectId   `json:"id,omitempty"          bson:"_id,omitempty"`
	User      bson.ObjectId   `json:"user,omitempty"        bson:"user"`
	ImageWebp string          `json:"-"                     bson:"image_webp"`
	ImageJpeg string          `json:"-"                     bson:"image_jpeg"`
	ImageUrl  string          `json:"image_url,omitempty"   bson:"-"`
	Time      time.Time       `json:"time"              bson:"time"`
	Photo     []bson.ObjectId `json:"photo,omitempty"       bson:"photo,omitempty"`
}

type Photo struct {
	Id            bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId `json:"user"                  bson:"user"`
	ImageWebp     string        `json:"-"                     bson:"image_webp"`
	ImageJpeg     string        `json:"-"                     bson:"image_jpeg"`
	ImageUrl      string        `json:"url"                   bson:"-"`
	ThumbnailWebp string        `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string        `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string        `json:"thumbnail_url"         bson:"-"`
	Description   string        `json:"description,omitempty" bson:"description,omitempty"`
	Time          time.Time     `json:"time"                  bson:"time"`
	// Comments    []Comment     `json:"comments,omitempty"    bson:"comments,omitempty"`
}

type Video struct {
	Id            bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId `json:"user"                  bson:"user"`
	VideoWebm     string        `json:"-"                     bson:"video_webm"`
	VideoMpeg     string        `json:"-"                     bson:"video_mpeg"`
	VideoUrl      string        `json:"url"                   bson:"-"`
	ThumbnailWebp string        `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string        `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string        `json:"thumbnail_url"         bson:"-"`
	Description   string        `json:"description,omitempty" bson:"description,omitempty"`
	Time          time.Time     `json:"time"                  bson:"time"`
	Duration      int64         `json:"duration"              bson:"duration"`
}

type Audio struct {
	Id          bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User        bson.ObjectId `json:"user"                  bson:"user"`
	AudioAac    string        `json:"-"                     bson:"audio_aac"`
	AudioOgg    string        `json:"-"                     bson:"audio_ogg"`
	AudioUrl    string        `json:"url"                   bson:"-"`
	Description string        `json:"description,omitempty" bson:"description,omitempty"`
	Time        time.Time     `json:"time"                  bson:"time"`
	Duration    int64         `json:"duration"              bson:"duration"`
}

type StripeItemRequest struct {
	Id   bson.ObjectId `json:"id"`
	Type string        `json:"type"`
}

type EmailConfirmationToken struct {
	Id    bson.ObjectId `bson:"_id"`
	User  bson.ObjectId `bson:"user"`
	Time  time.Time     `bson:"time"`
	Token string        `bson:"token"`
}

type PhoneConfirmationToken struct {
	Id    bson.ObjectId `bson:"_id"`
	User  bson.ObjectId `bson:"user"`
	Time  time.Time     `bson:"time"`
	Token string        `bson:"token"`
}

type ConfirmationMail struct {
	Destination string
	Mail        string
}

func (mail ConfirmationMail) From() string {
	return "noreply@" + mailDomain
}

func (mail ConfirmationMail) To() []string {
	return []string{mail.Destination}
}
func (mail ConfirmationMail) Cc() []string {
	return []string{}
}

func (mail ConfirmationMail) Bcc() []string {
	return []string{}
}
func (mail ConfirmationMail) Subject() string {
	return "confirmation"
}
func (mail ConfirmationMail) Html() string {
	return ""
}
func (mail ConfirmationMail) Text() string {
	return mail.Mail
}
func (mail ConfirmationMail) Headers() map[string]string {
	return map[string]string{}
}
func (mail ConfirmationMail) Options() map[string]string {
	return map[string]string{}
}
func (mail ConfirmationMail) Variables() map[string]string {
	return map[string]string{}
}

type Like struct {
	Id     bson.ObjectId `json:"id" bson:"_id"`
	User   bson.ObjectId `json:"user" bson:"user"`
	Target bson.ObjectId `json:"target" bson:"target"`
	Time   time.Time     `json:"time" bson:"time"`
}

type Transaction struct {
	Id    int           `bson:"_id"`
	Value int           `bson:"value"`
	User  bson.ObjectId `bson:"user"`
}
