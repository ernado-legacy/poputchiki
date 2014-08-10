package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

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
	Origin      string
}

func (mail ConfirmationMail) From() string {
	return "noreply@" + mail.Origin
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
