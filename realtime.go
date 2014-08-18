package main

import (
	"encoding/json"
	"fmt"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/models"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"github.com/riobard/go-mailgun"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	TOKEN_REDIS_KEY = "tokens"
	TOKEN_URL_PARM  = "token"
	REDIS_SEPARATOR = ":"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	REALTIME_REDIS_KEY   = "realtime"
	REALTIME_CHANNEL_KEY = "channel"
	RELT_BUFF_SIZE       = 100
	RELT_WS_BUFF_SIZE    = 10
	RELT_PING_RATE_MS    = 1000
)

type RealtimeRedis struct {
	pool  *redis.Pool
	chans map[bson.ObjectId]ReltChannel
}

func (realtime *RealtimeRedis) Conn() redis.Conn {
	return realtime.pool.Get()
}

func (r *RealtimeRedis) Push(update models.Update) error {
	conn := r.Conn()
	defer conn.Close()
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, update.Destination.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	eJson, err := json.Marshal(update)
	if err != nil {
		return err
	}
	_, err = conn.Do("PUBLISH", key, eJson)
	return err
}

func chackOrigin(r *http.Request) bool {
	return true
}

func (realtime *RealtimeRedis) RealtimeHandler(w http.ResponseWriter, r *http.Request, t *gotok.Token) (int, []byte) {
	u := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: chackOrigin}
	_, ok := w.(http.Hijacker)
	if !ok {
		log.Println("not ok")
	}
	conn, err := u.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	c := realtime.GetWSChannel(t.Id)

	connClosed := make(chan bool, 10)

	go func() {
		<-connClosed
		realtime.CloseWs(c)
		log.Println("connection closed")
	}()
	conn.WriteJSON(t)

	conn.SetPongHandler(func(s string) error {
		log.Println("pong")
		return nil
	})

	go func() {
		for {
			time.Sleep(time.Millisecond * RELT_PING_RATE_MS)
			err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second*5))
			if err != nil {
				connClosed <- true
				return
			}
		}
	}()

	for event := range c.channel {
		err := conn.WriteJSON(event)
		if err != nil {
			connClosed <- true
			return Render(ErrorBackend)
		}
	}
	return Render("ok")
}

func (realtime *RealtimeRedis) getChannel(id bson.ObjectId) chan models.Update {
	// creating new channel
	c := make(chan models.Update, RELT_BUFF_SIZE)
	conn := realtime.Conn()
	psc := redis.PubSubConn{}
	psc.Conn = conn
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, id.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	log.Println("starting listeting", key)
	psc.Subscribe(key)

	go func() {
		defer conn.Close()
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				e := new(models.Update)
				err := json.Unmarshal(v.Data, e)
				if err != nil {
					log.Println(err)
					return
				}
				c <- *e
			case error:
				log.Println(v)
				return
			}
		}
	}()
	return c
}

func pushAll(event models.Update, chans map[bson.ObjectId]chan models.Update) {
	for _, channel := range chans {
		channel <- event
	}
}

func (realtime *RealtimeRedis) GetReltChannel(id bson.ObjectId) ReltChannel {
	log.Println("getting realtime channel")
	c := ReltChannel{make(map[bson.ObjectId](chan models.Update)), realtime.getChannel(id)}
	go func() {
		for event := range c.events {
			go pushAll(event, c.chans)
		}
	}()
	return c
}

func (realtime *RealtimeRedis) GetWSChannel(id bson.ObjectId) ReltWSChannel {
	log.Println("getting websocket channel for", id.Hex())
	c := make(chan models.Update, RELT_WS_BUFF_SIZE)
	_, ok := realtime.chans[id]
	if !ok {
		log.Println("realtime channel not found, creating")
		realtime.chans[id] = realtime.GetReltChannel(id)
	}
	wsid := bson.NewObjectId()
	realtime.chans[id].chans[wsid] = c
	return ReltWSChannel{id: wsid, user: id, channel: c}
}

func (realtime *RealtimeRedis) CloseWs(c ReltWSChannel) {
	log.Println("closing realtime channel")
	delete(realtime.chans[c.user].chans, c.id)
}

type ReltWSChannel struct {
	id            bson.ObjectId
	user          bson.ObjectId
	channel       chan models.Update
	subscriptions []bson.ObjectId
}

type ReltChannel struct {
	chans  map[bson.ObjectId](chan models.Update)
	events chan models.Update
}

type RealtimeUpdater struct {
	db       models.DataBase
	realtime models.Updater
	email    models.Updater
}

type EmailUpdater struct {
	db     models.DataBase
	client *mailgun.Client
}

func (e *EmailUpdater) Push(update models.Update) error {
	u := e.db.Get(update.Destination)
	message := models.ConfirmationMail{}
	message.Destination = u.Email
	message.Origin = "noreply@" + mailDomain
	message.Mail = fmt.Sprintf("%+v", update)
	_, err := e.client.Send(message)
	return err
}

func (u *RealtimeUpdater) AutoHandle(user, destination bson.ObjectId, body interface{}) error {
	t := strings.ToLower(reflect.TypeOf(body).Name())
	return u.Handle(t, user, destination, body)
}

func (u *RealtimeUpdater) Handle(eventType string, user, destination bson.ObjectId, body interface{}) error {

	update := models.NewUpdate(destination, user, eventType, body)
	return u.Push(update)
}

func (u *RealtimeUpdater) Push(update models.Update) error {
	log.Println("handling", update.Type)
	target := u.db.Get(update.Destination)
	_, err := u.db.AddUpdateDirect(&update)
	if err != nil {
		log.Println(err)
		return err
	}
	if target.Online {
		u.realtime.Push(update)
	} else {
		subscription := models.GetEventType(update.Type, update.TargetType)
		subscribed, err := u.db.UserIsSubscribed(update.Destination, subscription)
		if err != nil {
			log.Println(err)
			return err
		}
		if subscribed && u.email != nil {
			log.Println("sending email")
			return u.email.Push(update)
		}
	}
	log.Println("handled")
	return nil
}

type autoUpdater struct {
	updater models.Updater
	token   *gotok.Token
}

func (a *autoUpdater) Push(destination bson.ObjectId, body interface{}) error {
	t := strings.ToLower(reflect.TypeOf(body).Name())
	return a.updater.Push(models.NewUpdate(destination, a.token.Id, t, body))
}

func AutoUpdaterWrapper(u models.Updater, t *gotok.Token, c martini.Context) {
	if t != nil && u != nil {
		var auto models.AutoUpdater
		auto = &autoUpdater{u, t}
		c.Map(auto)
	}
}
