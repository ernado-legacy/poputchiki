package main

import (
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
	Favorites  []bson.ObjectId `json:"favorites,omitempty" bson:"favorites,omitempty"`
	Guests     []bson.ObjectId `json:"guests,omitempty"    bson:"guests,omitempty"`
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
	u.Guests = nil
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
