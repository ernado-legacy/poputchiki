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

func getHash(password, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

var UserWritableFields = []string{"name", "email", "phone", "avatar", "birthday", "seasons",
	"city", "country", "weight", "growth", "destinations", "sex", "is_sponsor", "is_host",
}

const (
	FORM_EMAIL      = "email"
	FORM_PASSWORD   = "password"
	FORM_FIRSTNAME  = "firstname"
	FORM_SECONDNAME = "secondname"
	FORM_PHONE      = "phone"
)

type User struct {
	Id             bson.ObjectId   `json:"id"                     bson:"_id,omitempty"`
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

func UserFromForm(r *http.Request, salt string) *User {
	u := User{}
	u.Id = bson.NewObjectId()
	u.Email = r.FormValue(FORM_EMAIL)
	u.Password = getHash(r.FormValue(FORM_PASSWORD), salt)
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
	}
}

func (u *User) Prepare(adapter *weed.Adapter, db DataBase, webp WebpAccept) {
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
