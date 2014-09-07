package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"github.com/GeertJohan/go.rice"
	// "github.com/ernado/poputchiki/database"
	"github.com/ernado/poputchiki/models"
	"github.com/ernado/weed"
	_ "github.com/go-sql-driver/mysql"
	"github.com/riobard/go-mailgun"
	"gopkg.in/mgo.v2"
	"log"
	"text/template"
	// "time"
)

var (
	dbName  = "kafe_dev"
	dbHost  = "localhost"
	dbSalt  = "salt"
	weedUrl = "http://localhost:9333"
	db      *mgo.Database
	// db        models.DataBase
	adapter   *weed.Adapter
	templates *rice.Box
	mail      *mailgun.Client
)

// { "_id" : ObjectId("53455d9e83b270001310d512"), "date_created" : ISODate("2014-04-09T14:47:57.054Z"), "email" : "zhora_kornienko@mail.ru" }
type EmailEntry struct {
	Email string `bson:"email"`
}

func NewMail(src, origin, destination, subject string, data interface{}) (m models.Mail, err error) {
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

// func Process(sql.DB) {

// 	rows, err := mysqlDb.Query("SELECT email FROM users")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer rows.Close()
// 	for rows.Next() {
// 		var email sql.NullString
// 		if err := rows.Scan(&email); err != nil {
// 			log.Fatal(err)
// 		}
// 		userEmail := email.String
// 		u := db.GetUsername(userEmail)
// 		if u != nil {
// 			log.Printf("%s already in database", userEmail)
// 			continue
// 		}
// 		send(userEmail)
// 	}
// }

func Process() {
	var entries []EmailEntry
	log.Println("getting entry")
	if err := db.C("email_entry").Find(nil).All(&entries); err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		// log.Println(entry.Email)
		send(entry.Email)
	}
}

func send(email string) error {
	origin := fmt.Sprintf("Попутчики <%s>", "noreply@mail.poputchiki.ru")
	t, err := templates.String("invite.html")
	if err != nil {
		log.Fatal(err)
	}
	message, err := NewMail(t, origin, email, "Наконец-то запустился наш новый сайт!", nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = mail.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("%s processed", email)
	}
	return err
}

func main() {
	flag.StringVar(&dbName, "db.name", dbName, "Database name")
	flag.StringVar(&dbHost, "db.host", dbHost, "Mongo host")
	flag.StringVar(&dbSalt, "db.salt", dbSalt, "Database salt")
	flag.StringVar(&weedUrl, "weed.url", weedUrl, "WeedFs url")
	flag.Parse()
	session, err := mgo.Dial(dbHost)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(dbName)
	// db = database.New(dbName, dbSalt, time.Second, session)
	adapter = weed.NewAdapter(weedUrl)
	templates = rice.MustFindBox("templates")
	fmt.Println("poputchiki mailing system")
	log.Printf("connected to %s/%s", dbHost, dbName)
	log.Printf("weedfs url %s", weedUrl)
	log.Println("connecting to mysql")
	mail = mailgun.New("key-7520cy18i2ebmrrbs1bz4ivhua-ujtb6")
	mysqlDb, err := sql.Open("mysql", "root:root@/poputchiki")
	defer mysqlDb.Close()
	if err != nil {
		log.Fatal(err)
	}
	Process()
}
