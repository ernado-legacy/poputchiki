package main

import (
	"encoding/json"
	"fmt"
	"labix.org/v2/mgo/bson"
	"net/http"
	"time"
)

type User struct {
	Id         bson.ObjectId   `json:"id"                  bson:"_id"`
	FirstName  string          `json:"firstname"           bson:"firstname,omitempty"`
	SecondName string          `json:"secondname" 	       bson:"secondname,omitempty"`
	Email      string          `json:"email,omitempty"     bson:"email,omitempty"`
	Phone      string          `json:"phone,omitempty"     bson:"phone,omitempty"`
	Password   string          `json:"-"                   bson:"password"`
	Online     bool            `json:"online,omitempty"    bson:"online,omitempty"`
	Photo      Image           `json:"photo,omitempty"     bson:"photo",omitempty"`
	Balance    uint            `json:"balance,omitempty"   bson:"balance,omitempty"`
	LastAction time.Time       `json:"lastaction,omitempty"bson:"lastaction",omitempty"`
	Favorites  []bson.ObjectId `json:"favorites,omitempty" bson:"favorites,omitempty"`
	Blacklist  []bson.ObjectId `json:"blacklist,omitempty" bson:"blacklist,omitempty"`
	Countries  []string        `json:"countries,omitempty" bson:"countries,omitempty"`
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
	r.ParseForm()
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(u)
	if err == nil {
		return
	}

	email := r.FormValue(FORM_EMAIL)
	if email != BLANK {
		u.Email = email
	}
	password := r.FormValue(FORM_PASSWORD)
	if password != BLANK {
		u.Password = getHash(password)
	}
	phone := r.FormValue(FORM_PHONE)
	if phone != BLANK {
		u.Phone = phone
	}
	firstname := r.FormValue(FORM_FIRSTNAME)
	if firstname != BLANK {
		u.FirstName = firstname
	}
	secondname := r.FormValue(FORM_SECONDNAME)
	if secondname != BLANK {
		u.SecondName = secondname
	}
}

func (u *User) String() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.SecondName)
}

func (u *User) CleanPrivate() {
	u.Password = ""
	u.Phone = ""
	u.Email = ""
	u.Favorites = nil
	u.Blacklist = nil
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
	Id   bson.ObjectId `json:"id"          bson:"_id"`
	User bson.ObjectId `json:"user"        bson:"user"`
	Text string        `json:"text"        bson:"text"`
	Time time.Time     `json:"time"        bson:"time"`
}

type StatusUpdate struct {
	Id       bson.ObjectId `json:"id"          bson:"_id"`
	User     bson.ObjectId `json:"user"        bson:"user"`
	Time     time.Time     `json:"time"        bson:"time"`
	Text     string        `json:"text"        bson:"text"`
	Comments []Comment     `json:"comments"    bson:"comments"`
}

type StripeItem struct {
	Id        bson.ObjectId `json:"id"                  bson:"_id"`
	User      bson.ObjectId `json:"user"                bson:"user"`
	Countries []string      `json:"countries,omitempty" bson:"countries,omitempty"`
	Time      time.Time     `json:"time"                bson:"time"`
}

type Image struct {
	Id  bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Fid string        `json:"fid"          bson:"fid"`
	Url string        `json:"url"          bson:"url,omitempty"`
}

//{"count":1,"fid":"3,01637037d6","url":"127.0.0.1:8080","publicUrl":"localhost:8080"}
type WeedAssign struct {
	Count     int    `json:"count"`
	Fid       string `json:"fid"`
	Url       string `json:"url"`
	PublicUrl string `json:"publicUrl"`
}
