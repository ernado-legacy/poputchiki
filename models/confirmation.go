package models

import (
	"bytes"
	"gopkg.in/mgo.v2/bson"
	"html/template"
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

type Mail struct {
	Destination string
	Mail        string
	Title       string
	Origin      string
}

func (mail ConfirmationMail) From() string {
	return mail.Origin
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

func (mail Mail) From() string {
	return mail.Origin
}

func (mail Mail) To() []string {
	return []string{mail.Destination}
}
func (mail Mail) Cc() []string {
	return []string{}
}

func (mail Mail) Bcc() []string {
	return []string{}
}
func (mail Mail) Subject() string {
	return mail.Title
}
func (mail Mail) Html() string {
	return mail.Mail
}
func (mail Mail) Text() string {
	return ""
}
func (mail Mail) Headers() map[string]string {
	return map[string]string{}
}
func (mail Mail) Options() map[string]string {
	return map[string]string{}
}
func (mail Mail) Variables() map[string]string {
	return map[string]string{}
}

func NewMail(src, origin, destination, subject string, data interface{}) (m Mail, err error) {
	m.Origin = origin
	m.Title = subject
	m.Destination = destination
	t, err := template.New("template").Parse(src)
	if err != nil {
		return
	}
	buff := new(bytes.Buffer)
	if err = t.Execute(buff, data); err != nil {
		return
	}
	m.Mail = buff.String()
	return
}
