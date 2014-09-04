package models

import (
	"bytes"
	"github.com/GeertJohan/go.rice"
	"github.com/riobard/go-mailgun"
	"gopkg.in/mgo.v2/bson"
	"log"
	"text/template"
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
	log.Printf("[email] %+v", data)
	if err = t.Execute(buff, data); err != nil {
		log.Println("[email]", err)
		return
	}
	m.Mail = buff.String()
	log.Println("[email]", m.Mail)
	return
}

type MailDispatcher struct {
	box    *rice.Box
	origin string
	client *mailgun.Client
	db     DataBase
}

type MailHtmlSender interface {
	Send(template string, destination bson.ObjectId, subject string, data interface{}) error
	SendTo(template, destination, subject string, data interface{}) error
}

func GetMailDispatcher(box *rice.Box, email string, client *mailgun.Client, db DataBase) MailHtmlSender {
	return MailDispatcher{box, email, client, db}
}

func (dispatcher MailDispatcher) Send(template string, destination bson.ObjectId, subject string, data interface{}) error {
	destinationEmail := dispatcher.db.Get(destination).Email
	return dispatcher.SendTo(template, destinationEmail, subject, data)
}

func (dispatcher MailDispatcher) SendTo(template, destination, subject string, data interface{}) error {
	src, err := dispatcher.box.String(template)
	if err != nil {
		return err
	}
	m, err := NewMail(src, dispatcher.origin, destination, subject, data)
	if err != nil {
		return err
	}
	_, err = dispatcher.client.Send(m)
	return err
}
