package models

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/ernado/weed"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"time"
)

var UserWritableFields = []string{"name", "email", "phone", "avatar", "birthday", "seasons",
	"city", "country", "weight", "growth", "destinations", "sex", "is_sponsor", "is_host",
	"likings_sex", "likings_destinations", "likings_seasons", "about",
}

const (
	FormEmail      = "email"     // email field
	FormPassword   = "password"  // password field
	FormFirstname  = "firstname" // user name field
	FormPhone      = "phone"
	femaleNoAvater = "http://poputchiki/static/img/defaultavatars/no_avatar_f.gif"
	maleNoAvatar   = "http://poputchiki/static/img/defaultavatars/no_avatar_m.gif"
)

// UserInfo additional user information
type UserInfo struct {
	Id    bson.ObjectId `json:"id"     bson:"_id"`
	About string        `json:"about"  bson:"about"`
}

type Guest struct {
	Id    bson.ObjectId `json:"id"    bson:"_id"`
	User  bson.ObjectId `json:"user"  bson:"user"`
	Guest bson.ObjectId `json:"guest" bson:"guest"`
	Time  time.Time     `json:"time"  bson:"time"`
}

type User struct {
	Id                  bson.ObjectId   `json:"id"                     bson:"_id,omitempty"`
	Name                string          `json:"name,omitempty"         bson:"name,omitempty"`
	Sex                 string          `json:"sex,omitempty"          bson:"sex,omitempty"`
	Email               string          `json:"email,omitempty"        bson:"email,omitempty"`
	EmailConfirmed      bool            `json:"email_confirmed"        bson:"email_confirmed"`
	Phone               string          `json:"phone,omitempty"        bson:"phone,omitempty"`
	PhoneConfirmed      bool            `json:"phone_confirmed"        bson:"phone_confirmed"`
	IsSponsor           bool            `json:"is_sponsor"             bson:"is_sponsor"`
	IsHost              bool            `json:"is_host"                bson:"is_host"`
	Password            string          `json:"password,omitempty"     bson:"password"`
	Online              bool            `json:"online"                 bson:"online"`
	AvatarUrl           string          `json:"avatar_url,omitempty"   bson:"-"`
	Avatar              bson.ObjectId   `json:"avatar,omitempty"       bson:"avatar,omitempty"`
	AvatarWebp          string          `json:"-"                      bson:"image_webp,omitempty"`
	AvatarJpeg          string          `json:"-"                      bson:"image_jpeg,omitempty"`
	Balance             uint            `json:"balance,omitempty"      bson:"balance,omitempty"`
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
	StatusUpdate        time.Time       `json:"-"                      bson:"statusupdate,omitempty"`
	Favorites           []bson.ObjectId `json:"favorites,omitempty"    bson:"favorites,omitempty"`
	Blacklist           []bson.ObjectId `json:"blacklist,omitempty"    bson:"blacklist,omitempty"`
	Countries           []string        `json:"countries,omitempty"    bson:"countries,omitempty"`
	LikingsSex          string          `json:"likings_sex,omitempty"          bson:"likings_sex,omitempty"`
	LikingsDestinations []string        `json:"likings_destinations,omitempty" bson:"likings_destinations,omitempty"`
	LikingsSeasons      []string        `json:"likings_seasons,omitempty"      bson:"likings_seasons,omitempty"`
	LikingsCountry      string          `json:"likings_country,omitempty"      bson:"likings_country,omitempty"`
	LikingsCity         string          `json:"likings_city,omitempty"      bson:"likings_city,omitempty"`
}

func getHash(password, salt string) string {
	log.Printf("sha256(%s,%s)", password, salt)
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func UserFromForm(r *http.Request, salt string) *User {
	u := User{}
	tUser := &User{}
	parser := NewParser(r)
	err := parser.Parse(tUser)
	log.Println(err, tUser)
	u.Id = bson.NewObjectId()
	u.Email = tUser.Email
	u.Password = getHash(tUser.Password, salt)
	u.Phone = tUser.Phone
	u.Name = tUser.Name
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

func (u *User) Prepare(adapter *weed.Adapter, db DataBase, webp WebpAccept) {
	u.SetAvatarUrl(adapter, db, webp)
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
