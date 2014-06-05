package main

import (
	"encoding/json"
	"github.com/ginuerzh/weedo"
	"labix.org/v2/mgo/bson"
	"net/http"

	"time"
)

type User struct {
	Id         bson.ObjectId   `json:"id"                     bson:"_id"`
	FirstName  string          `json:"firstname"              bson:"firstname,omitempty"`
	SecondName string          `json:"secondname"             bson:"secondname,omitempty"`
	Email      string          `json:"email,omitempty"        bson:"email,omitempty"`
	Phone      string          `json:"phone,omitempty"        bson:"phone,omitempty"`
	Password   string          `json:"-"                      bson:"password"`
	Online     bool            `json:"online,omitempty"       bson:"online,omitempty"`
	AvatarUrl  string          `json:"avatar_url,omitempty"   bson:"-"`
	Avatar     bson.ObjectId   `json:"avatar,omitempty"       bson:"avatar,omitempty"`
	Balance    uint            `json:"balance,omitempty"      bson:"balance,omitempty"`
	Age        uint            `json:"age,omitempty"          bson:"age,omitempty"`
	LastAction time.Time       `json:"last_action,omitempty"  bson:"last_action,omitempty"`
	Favorites  []bson.ObjectId `json:"favorites,omitempty"    bson:"favorites,omitempty"`
	Blacklist  []bson.ObjectId `json:"blacklist,omitempty"    bson:"blacklist,omitempty"`
	Countries  []string        `json:"countries,omitempty"    bson:"countries,omitempty"`
}

// additional user info
type UserInfo struct {
	Id           bson.ObjectId `json:"id"                     bson:"_id"`
	Weight       uint          `json:"weight,omitempty"       bson:"weight,omitempty"`
	Growth       uint          `json:"growth,omitempty"       bson:"growth,omitempty"`
	Destinations []string      `json:"destinations,omitempty" bson:"destinations,omitempty"`
	Seasons      []string      `json:"seasons,omitempty"      bson:"seasons,omitempty"`
}

type Pagination struct {
	Number int
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

// func (v *Video) SetUrl(c *weedo.Client)

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
	Id           bson.ObjectId `json:"id,omitempty"          bson:"_id,omitempty"`
	User         bson.ObjectId `json:"user"                  bson:"user"`
	AudioMp3     string        `json:"-"                     bson:"audio_mp3"`
	AudioOgg     string        `json:"-"                     bson:"audio_ogg"`
	AudioUrl     string        `json:"url"                   bson:"-"`
	ThumbnailUrl string        `json:"thumbnail_url"         bson:"-"`
	Description  string        `json:"description,omitempty" bson:"description,omitempty"`
	Time         time.Time     `json:"time"                  bson:"time"`
	Duration     int64         `json:"duration"              bson:"duration"`
}
