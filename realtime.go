package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/models"
	. "github.com/ernado/poputchiki/models"
	"github.com/ernado/weed"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"github.com/riobard/go-mailgun"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"net/url"
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
	REALTIME_GOLBAL      = "global"
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
	log.Println("[realtime] pushing", update)
	conn := r.Conn()
	defer conn.Close()
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, update.Destination.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	eJson, err := json.Marshal(update)
	if err != nil {
		return err
	}
	_, err = conn.Do("PUBLISH", key, eJson)
	if err == nil {
		log.Println("[realtime] pushed", update)
	}
	return err
}

func (r *RealtimeRedis) PushGlobal(update models.Update) error {
	log.Println("[realtime] pushing global", update)
	conn := r.Conn()
	defer conn.Close()
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, REALTIME_GOLBAL}
	key := strings.Join(args, REDIS_SEPARATOR)
	eJson, err := json.Marshal(update)
	if err != nil {
		return err
	}
	_, err = conn.Do("PUBLISH", key, eJson)
	if err == nil {
		log.Println("[realtime] pushed global", update)
	}
	return err
}

func chackOrigin(r *http.Request) bool {
	return true
}

func (realtime *RealtimeRedis) RealtimeHandler(w http.ResponseWriter, context Context) (int, []byte) {
	r := context.Request
	t := context.Token
	admin := context.IsAdmin
	u := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: chackOrigin}
	_, ok := w.(http.Hijacker)
	if !ok {
		log.Println("not ok")
	}
	conn, err := u.Upgrade(w, r, nil)
	if err != nil {
		return Render(BackendError(err))
	}

	q := r.URL.Query()
	var channels []ReltWSChannel
	channel := make(chan models.Update)
	var targets []bson.ObjectId
	if admin && q.Get("id") != "" {
		ids := strings.Split(q.Get("id"), ",")
		for _, target := range ids {
			if !bson.IsObjectIdHex(target) {
				return Render(ErrorBadRequest)
			}
			targets = append(targets, bson.ObjectIdHex(target))
		}
	} else {
		targets = append(targets, t.Id)
	}

	for _, target := range targets {
		c := realtime.GetWSChannel(target)
		channels = append(channels, c)
		go func() {
			for event := range c.channel {
				channel <- event
			}
		}()
	}

	connClosed := make(chan bool, 10)

	go func() {
		<-connClosed
		for _, c := range channels {
			realtime.CloseWs(c)
		}
		log.Println("connection closed")
	}()

	conn.WriteJSON(models.NewUpdate(t.Id, t.Id, "token", t))
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

	process := func(event models.Update) error {
		log.Println("[realtime] recieved event for", event.Destination.Hex())
		log.Printf("%+v", event)
		if err := event.Prepare(context); err != nil {
			return err
		}
		if err := conn.WriteJSON(event); err != nil {
			connClosed <- true
			return err
		}
		return nil
	}

	for event := range channel {
		if err := process(event); err != nil {
			return Render(BackendError(err))
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
	args = []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, REALTIME_GOLBAL}
	psc.Subscribe(strings.Join(args, REDIS_SEPARATOR))

	go func() {
		defer conn.Close()
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				log.Println("[realtime] recieved message")
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
	push     models.Updater
}

type EmailUpdater struct {
	db        models.DataBase
	client    *mailgun.Client
	templates *rice.Box
	adapter   *weed.Adapter
}

type PushNotificationsUpdater struct {
	db      models.DataBase
	adapter *weed.Adapter
}

func (e *PushNotificationsUpdater) Push(update models.Update) error {
	user := e.db.Get(update.Destination)
	if len(user.IOsTokens) == 0 && len(user.AndroidTokens) == 0 {
		log.Println("[updates]", "no tokens")
		return nil
	}
	u := url.URL{}
	u.Host = "cydev.ru"
	u.Scheme = "https"
	u.Path = fmt.Sprintf("/%s/push", "poputchiki")
	q := u.Query()
	context := Context{}
	context.DB = e.db
	context.Storage = e.adapter
	if err := update.Prepare(context); err != nil {
		log.Println("[email]", err)
	}
	q.Add("message", update.Theme())
	for _, token := range user.IOsTokens {
		q.Add("appleid", token)
	}
	for _, token := range user.AndroidTokens {
		q.Add("googleid", token)
	}
	u.RawQuery = q.Encode()
	resp, err := http.Post(u.String(), "text/html", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Bad code %s!=200", resp.StatusCode))
	}
	return nil
}

func (e *EmailUpdater) GetTemplate(update models.Update) (template string, err error) {
	filename := fmt.Sprintf("%s.html", update.Type)
	log.Println("[email]", "template", filename)
	return e.templates.String(filename)
}

func (e *EmailUpdater) Push(update models.Update) error {
	log.Println("[email]", "pushing")
	u := e.db.Get(update.Destination)
	context := Context{}
	context.DB = e.db
	context.Storage = e.adapter
	u.Prepare(context)
	if err := update.Prepare(context); err != nil {
		log.Println("[email]", err)
	}
	template, err := e.GetTemplate(update)
	if err != nil {
		log.Println("[email] template error", err)
		return err
	}
	email := fmt.Sprintf("Попутчики <%s>", "noreply@"+mailDomain)
	message, err := models.NewMail(template, email, u.Email, update.Theme(), update)
	if err != nil {
		return err
	}
	_, err = e.client.Send(message)
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
	log.Println("[updates]", "handling", update)
	target := u.db.Get(update.Destination)
	dublicate, err := u.db.IsUpdateDublicate(update.User, update.Destination, update.Type, DublicateUpdatesTimeout)
	if err != nil {
		return err
	}
	if dublicate && update.Type == models.UpdateGuests {
		log.Println("[updates]", "dublicate")
		return nil
	}
	_, err = u.db.AddUpdateDirect(&update)
	if err != nil {
		log.Println("[updates]", "realtime error", err)
		return err
	}

	if err := u.realtime.Push(update); err != nil {
		log.Println("[updates]", "realtime error", err)
		return err
	}

	if dublicate {
		log.Println("[updates]", "dublicate")
		return nil
	}
	if err := u.push.Push(update); err != nil {
		log.Println("[updates] push", err)
	}
	if !target.Online {
		log.Println("[updates]", "user offline")
		subscription := models.GetEventType(update.Type, update.Target)
		subscribed, err := u.db.UserIsSubscribed(update.Destination, subscription)
		if err != nil {
			log.Println(err)
			return err
		}
		subscribed = true
		if subscribed && u.email != nil {
			log.Println("[updates]", "sending email")
			return u.email.Push(update)
		} else {
			log.Println("[updates]", "not subscribed")
		}
	}
	log.Println("[updates]", "handled", update)
	return nil
}

type autoUpdater struct {
	updater models.Updater
	token   *gotok.Token
}

func (a *autoUpdater) Push(destination bson.ObjectId, body interface{}) error {
	t := strings.ToLower(reflect.TypeOf(body).Elem().Name())
	if t == "message" {
		t = "messages"
	}
	if t == "invite" {
		t = "invites"
	}
	return a.updater.Push(models.NewUpdate(destination, a.token.Id, t, body))
}

func AutoUpdaterWrapper(u models.Updater, t *gotok.Token, c martini.Context) {
	if t != nil && u != nil {
		var auto models.AutoUpdater
		auto = &autoUpdater{u, t}
		c.Map(auto)
	}
}
