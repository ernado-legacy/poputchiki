package models

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/ernado/weed"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"strings"
	"time"
)

var UserWritableFields = []string{"name", "email", "phone", "avatar", "birthday", "seasons",
	"city", "country", "weight", "growth", "destinations", "sex", "is_sponsor", "is_host",
	"likings_sex", "likings_destinations", "likings_seasons", "likings_country", "likings_city",
	"about", "location", "likings_age_min", "likings_age_max", "password", "invisible", "subscriptions",
}

const (
	FormEmail               = "email"     // email field
	FormPassword            = "password"  // password field
	FormFirstname           = "firstname" // user name field
	FormPhone               = "phone"
	femaleNoAvater          = "https://poputchiki.cydev.ru/static/img/defaultavatars/no_avatar_f.gif"
	maleNoAvatar            = "https://poputchiki.cydev.ru/static/img/defaultavatars/no_avatar_m.gif"
	SubscriptionLikesPhoto  = "likes_photo"
	SubscriptionLikesStatus = "likes_status"
	SubscriptionMessages    = "messages"
	SubscriptionInvites     = "invites"
	SubscriptionGuests      = "guests"
	SubscriptionNews        = "news"
)

var (
	Subscriptions = []string{SubscriptionLikesPhoto, SubscriptionLikesStatus, SubscriptionMessages,
		SubscriptionInvites, SubscriptionGuests, SubscriptionNews}
)

// UserInfo additional user information
type UserInfo struct {
	Id    bson.ObjectId `json:"id"     bson:"_id"`
	About string        `json:"about"  bson:"about"`
}

type Guest struct {
	Id    bson.ObjectId `json:"id"    bson:"_id,omitempty"`
	User  bson.ObjectId `json:"user"  bson:"user"`
	Guest bson.ObjectId `json:"guest" bson:"guest"`
	Time  time.Time     `json:"time"  bson:"time"`
}

type User struct {
	Id                  bson.ObjectId   `json:"id,omitempty"           bson:"_id,omitempty"`
	Name                string          `json:"name,omitempty"         bson:"name,omitempty"`
	Sex                 string          `json:"sex,omitempty"          bson:"sex,omitempty"`
	Email               string          `json:"email,omitempty"        bson:"email,omitempty"`
	EmailConfirmed      bool            `json:"email_confirmed"        bson:"email_confirmed"`
	Phone               string          `json:"phone,omitempty"        bson:"phone,omitempty"`
	PhoneConfirmed      bool            `json:"phone_confirmed"        bson:"phone_confirmed"`
	IsSponsor           bool            `json:"is_sponsor,omitempty"   bson:"is_sponsor"`
	IsHost              bool            `json:"is_host,omitempty"      bson:"is_host"`
	Password            string          `json:"password,omitempty"     bson:"password"`
	Online              bool            `json:"online"                 bson:"online"`
	AvatarUrl           string          `json:"avatar_url,omitempty"   bson:"-"`
	Avatar              bson.ObjectId   `json:"avatar,omitempty"       bson:"avatar,omitempty"`
	AvatarWebp          string          `json:"-"                      bson:"image_webp,omitempty"`
	AvatarJpeg          string          `json:"-"                      bson:"image_jpeg,omitempty"`
	AudioUrl            string          `json:"audio_url,omitempty"    bson:"-"`
	Audio               bson.ObjectId   `json:"audio,omitempty"        bson:"audio,omitempty"`
	AudioAAC            string          `json:"-"                      bson:"audio_aac,omitempty"`
	AudioOGG            string          `json:"-"                      bson:"audio_ogg,omitempty"`
	Balance             uint            `json:"balance"                bson:"balance,omitempty"`
	About               string          `json:"about,omitempty"        bson:"about,omitempty"`
	Age                 int             `json:"age,omitempty"          bson:"-"`
	Birthday            time.Time       `json:"birthday,omitempty"     bson:"birthday,omitempty"`
	City                string          `json:"city,omitempty"         bson:"city,omitempty"`
	Country             string          `json:"country,omitempty"      bson:"country,omitempty"`
	Weight              uint            `json:"weight,omitempty"       bson:"weight,omitempty"`
	Growth              uint            `json:"growth,omitempty"       bson:"growth,omitempty"`
	Destinations        []string        `json:"destinations,omitempty" bson:"destinations,omitempty"`
	Seasons             []string        `json:"seasons,omitempty"      bson:"seasons,omitempty"`
	LastAction          time.Time       `json:"last_action,omitempty"  bson:"lastaction,omitempty"`
	Status              string          `json:"status,omitempty"       bson:"status"`
	StatusUpdate        time.Time       `json:"status_time,omitempty"  bson:"statusupdate,omitempty"`
	Favorites           []bson.ObjectId `json:"favorites,omitempty"    bson:"favorites,omitempty"`
	Blacklist           []bson.ObjectId `json:"blacklist,omitempty"    bson:"blacklist,omitempty"`
	Countries           []string        `json:"countries,omitempty"    bson:"countries,omitempty"`
	LikingsSex          string          `json:"likings_sex,omitempty"          bson:"likings_sex,omitempty"`
	LikingsDestinations []string        `json:"likings_destinations,omitempty" bson:"likings_destinations,omitempty"`
	LikingsSeasons      []string        `json:"likings_seasons,omitempty"      bson:"likings_seasons,omitempty"`
	LikingsCountry      string          `json:"likings_country,omitempty"      bson:"likings_country,omitempty"`
	LikingsCity         string          `json:"likings_city,omitempty"         bson:"likings_city,omitempty"`
	LikingsAgeMin       int             `json:"likings_age_min,omitempty"      bson:"likings_age_min,omitempty"`
	LikingsAgeMax       int             `json:"likings_age_max,omitempty"      bson:"likings_age_max,omitempty"`
	IsAdmin             bool            `json:"-"                      bson:"is_admin"`
	IsFavorite          bool            `json:"is_favourite"           bson:"-"`
	Location            []float64       `json:"location,omitempty"     bson:"location"`
	Invisible           bool            `json:"invisible,omitempty"    bson:"invisible"`
	Vip                 bool            `json:"vip"                    bson:"vip,omitempty"`
	VipTill             time.Time       `json:"vip_till"               bson:"vip_till"`
	Rating              float64         `json:"rating"                 bson:"rating"`
	Subscriptions       []string        `json:"subscriptions,omitempty"bson:"subscriptions"`
	Registered          time.Time       `json:"registered,omitempty"   bson:"registered"`
	Orientation         string          `json:"orientation" bson:"orientation,omitempty"`
	Relations           string          `json:"relations" bson:"relations,omitempty"`
	Children            string          `json:"children" bson:"children,omitempty"`
	Education           string          `json:"education" bson:"education,omitempty"`
	AttitudeSmoking     string          `json:"attitude_to_smoking" bson:"attitude_to_smoking,omitempty"`
	AttitudeAlcohol     string          `json:"attitude_to_alcohol" bson:"attitude_to_alcohol,omitempty"`
	Wealth              string          `json:"wealth" bson:"wealth,omitempty"`
	Accommodation       string          `json:"accommodation" bson:"accommodation,omitempty"`
}

type GuestUser struct {
	User `bson:"-"`
	Time time.Time `json:"time"  bson:"time"`
}

func getHash(password, salt string) string {
	// log.Printf("sha256(%s,%s)", password, salt)
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func UserFromForm(r *http.Request, salt string) *User {
	u := new(User)
	tUser := new(User)
	parser := NewParser(r)
	parser.Parse(tUser)
	u.Id = bson.NewObjectId()
	u.Email = strings.ToLower(tUser.Email)
	u.Password = getHash(tUser.Password, salt)
	u.Phone = tUser.Phone
	u.Name = tUser.Name
	return u
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

func (u *User) SetAvatarUrl(adapter *weed.Adapter, db DataBase, webp WebpAccept) {
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
	} else {
		if u.Sex == SexFemale {
			u.AvatarUrl = femaleNoAvater
		} else {
			u.AvatarUrl = maleNoAvatar
		}
	}
}

// diff in years between two times
func diff(t1, t2 time.Time) (years int) {
	t2 = t2.AddDate(0, 0, 1) // advance t2 to make the range inclusive
	for t1.AddDate(years, 0, 0).Before(t2) {
		years++
	}
	years--
	return years
}

func (u *User) SetIsFavorite(user *User) {
	for _, v := range user.Favorites {
		if v == u.Id {
			u.IsFavorite = true
		}
	}
}

type Users []*User

func (u Users) SetIsFavorite(user *User) {
	for _, v := range u {
		v.SetIsFavorite(user)
	}
}

func (u Users) Prepare(adapter *weed.Adapter, db DataBase, webp WebpAccept, audio AudioAccept, user *User) {
	u.SetIsFavorite(user)
	for _, v := range u {
		v.Prepare(adapter, db, webp, audio)
		v.CleanPrivate()
	}
}

func (u *User) Prepare(adapter *weed.Adapter, db DataBase, webp WebpAccept, audio AudioAccept) {
	u.Password = ""
	u.SetAvatarUrl(adapter, db, webp)

	if u.Audio != "" {
		u.AudioUrl, _ = adapter.GetUrl(u.AudioAAC)
		if audio == AaOgg {
			u.AudioUrl, _ = adapter.GetUrl(u.AudioOGG)
		}
	}

	if len(u.Favorites) == 0 {
		u.Favorites = []bson.ObjectId{}
	}
	if len(u.Blacklist) == 0 {
		u.Blacklist = []bson.ObjectId{}
	}

	now := time.Now()
	defer func() {
		if r := recover(); r != nil {
			u.Birthday = time.Unix(0, 0)
		}
	}()
	if u.Birthday.Unix() != 0 {
		u.Age = diff(u.Birthday, now)
	}

}
