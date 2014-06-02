package main

import (
	"encoding/json"
	"github.com/ginuerzh/weedo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"time"
)

type User struct {
	Id         bson.ObjectId   `json:"id"                   bson:"_id"`
	FirstName  string          `json:"firstname"            bson:"firstname,omitempty"`
	SecondName string          `json:"secondname" 	        bson:"secondname,omitempty"`
	Email      string          `json:"email,omitempty"      bson:"email,omitempty"`
	Phone      string          `json:"phone,omitempty"      bson:"phone,omitempty"`
	Password   string          `json:"-"                    bson:"password"`
	Online     bool            `json:"online,omitempty"     bson:"online,omitempty"`
	AvatarUrl  string          `json:"avatar_url,omitempty" bson:"-"`
	Avatar     bson.ObjectId   `json:"avatar,omitempty"     bson:"avatar,omitempty"`
	Balance    uint            `json:"balance,omitempty"    bson:"balance,omitempty"`
	LastAction time.Time       `json:"lastaction,omitempty" bson:"lastaction",omitempty"`
	Favorites  []bson.ObjectId `json:"favorites,omitempty"  bson:"favorites,omitempty"`
	Blacklist  []bson.ObjectId `json:"blacklist,omitempty"  bson:"blacklist,omitempty"`
	Countries  []string        `json:"countries,omitempty"  bson:"countries,omitempty"`
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
	u.FirstName = r.FormValue(FORM_FIRSTNAME)
	u.SecondName = r.FormValue(FORM_SECONDNAME)
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

func (u *User) SetAvatarUrl(c *weedo.Client, db UserDB, webp WebpAccept) {
	photo, err := db.GetPhoto(u.Avatar)
	if err == nil {
		suffix := ".jpg"
		fid := photo.ThumbnailJpeg
		if webp {
			fid = photo.ThumbnailWebp
			suffix = ".webp"
		}
		url, _, _ := c.GetUrl(fid)
		u.AvatarUrl = url + suffix
	}
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
	Progress float32 `json:"progress"`
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

type StatusUpdate struct {
	Id       bson.ObjectId `json:"id"       bson:"_id"`
	User     bson.ObjectId `json:"user"     bson:"user"`
	Time     time.Time     `json:"time"     bson:"time"`
	Text     string        `json:"text"     bson:"text"`
	Comments []Comment     `json:"comments" bson:"comments"`
}

type StripeItem struct {
	Id        bson.ObjectId `json:"id"                  bson:"_id"`
	User      bson.ObjectId `json:"user"                bson:"user"`
	Image     Image         `json:"image"               bson:"image"`
	Countries []string      `json:"countries,omitempty" bson:"countries,omitempty"`
	Time      time.Time     `json:"time"                bson:"time"`
}

type File struct {
	Id   bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Fid  string        `json:"fid"          bson:"fid"`
	User bson.ObjectId `json:"user"         bson:"user"`
	Time time.Time     `json:"time"         bson:"time"`
	Type string        `json:"type"         bson:"type"`
	Size int64         `json:"size"         bson:"size"`
}

type Image struct {
	Id  bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Fid string        `json:"fid"          bson:"fid"`
	Url string        `json:"url"          bson:"url,omitempty"`
}

type Album struct {
	Id        bson.ObjectId   `json:"id,omitempty"          bson:"_id,omitempty"`
	User      bson.ObjectId   `json:"user,omitempty""                  bson:"user"`
	ImageWebp string          `json:"-"                     bson:"image_webp"`
	ImageJpeg string          `json:"-"                     bson:"image_jpeg"`
	ImageUrl  string          `json:"image_url,omitempty"   bson:"-"`
	Time      time.Time       `json:"time"         		    bson:"time"`
	Photo     []bson.ObjectId `json:"photo,omitempty"       bson:"photo,omitempty"`
}

type Photo struct {
	Id            bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User          bson.ObjectId `json:"user"                  bson:"user"`
	ImageWebp     string        `json:"-"                     bson:"image_webp"`
	ImageJpeg     string        `json:"-"                     bson:"image_jpeg"`
	ImageUrl      string        `json:"image_url"             bson:"-"`
	ThumbnailWebp string        `json:"-"                     bson:"thumbnail_webp"`
	ThumbnailJpeg string        `json:"-"                     bson:"thumbnail_jpeg"`
	ThumbnailUrl  string        `json:"thumbnail_url"         bson:"-"`
	Description   string        `json:"description,omitempty" bson:"description,omitempty"`
	Time          time.Time     `json:"time"         		  bson:"time"`

	// Comments    []Comment     `json:"comments,omitempty"    bson:"comments,omitempty"`
}
