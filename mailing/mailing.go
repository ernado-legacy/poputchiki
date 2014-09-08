package main

import (
	"bytes"
	// "database/sql"
	"flag"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/database"
	"github.com/ernado/poputchiki/models"
	"github.com/ernado/weed"
	// _ "github.com/go-sql-driver/mysql"
	"github.com/riobard/go-mailgun"
	"gopkg.in/mgo.v2"
	"log"
	// "strings"
	"text/template"
	"time"
)

var (
	dbName  = "kafe_dev"
	dbHost  = "localhost"
	dbSalt  = "salt"
	weedUrl = "http://localhost:9333"
	// db      *mgo.Database
	db        *database.DB
	adapter   *weed.Adapter
	templates *rice.Box
	tokens    gotok.Storage
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
	log.Println("getting users")
	user := db.GetUsername("ernado@yandex.ru")
	if user == nil {
		log.Println("nil")
	}
	send(user)

	// users, count, err := db.Search(&models.SearchQuery{}, models.Pagination{Count: 5000})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("found %d users", count)
	// for _, user := range users {
	// 	if len(user.Email) < 0 {
	// 		continue
	// 	}
	// 	if strings.Index(user.Email, "test") != -1 {
	// 		continue
	// 	}
	// 	if err := send(entry.Email); err != nil {
	// 		log.Println(err)
	// 	}
	// }
}

func send(u *models.User) error {
	email := u.Email
	origin := fmt.Sprintf("Попутчики <%s>", "noreply@mail.poputchiki.ru")
	type Data struct {
		Url   string
		Email string
	}
	token := db.NewConfirmationToken(u.Id)
	data := Data{"http://poputchiki.ru/api/confirm/email/" + token.Token, u.Email}
	t, err := templates.String("registration.html")
	if err != nil {
		log.Fatal(err)
	}
	message, err := NewMail(t, origin, email, "Наконец-то запустился наш новый сайт!", data)
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
	// db = session.DB(dbName)
	db = database.New(dbName, dbSalt, time.Second, session)
	tokenCollection := session.DB(dbName).C("tokens")
	tokens = gotok.New(tokenCollection)
	adapter = weed.NewAdapter(weedUrl)
	templates = rice.MustFindBox("templates")
	fmt.Println("poputchiki mailing system")
	log.Printf("connected to %s/%s", dbHost, dbName)
	log.Printf("weedfs url %s", weedUrl)
	log.Println("connecting to mysql")
	mail = mailgun.New("key-7520cy18i2ebmrrbs1bz4ivhua-ujtb6")
	// mysqlDb, err := sql.Open("mysql", "root:root@/poputchiki")
	// defer mysqlDb.Close()
	if err != nil {
		log.Fatal(err)
	}
	Process()
}
