package main

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
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

func (realtime *RealtimeRedis) Push(id bson.ObjectId, event interface{}) error {
	conn := realtime.Conn()
	defer conn.Close()

	t := strings.ToLower(reflect.TypeOf(event).Name())
	e := RealtimeEvent{t, event, time.Now()}
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, id.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	eJson, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
		return err
	}
	// log.Println("pushing event", string(eJson), key)
	_, err = conn.Do("PUBLISH", key, eJson)
	if err != nil {
		log.Println(err)
	}
	return err
}

func chackOrigin(r *http.Request) bool {
	return true
}

func (realtime *RealtimeRedis) RealtimeHandler(w http.ResponseWriter, r *http.Request, t *Token) (int, []byte) {
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

func (realtime *RealtimeRedis) getChannel(id bson.ObjectId) chan RealtimeEvent {
	// creating new channel
	c := make(chan RealtimeEvent, RELT_BUFF_SIZE)

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
				e := RealtimeEvent{}
				err := json.Unmarshal(v.Data, &e)
				if err != nil {
					log.Println(err)
					return
				}
				c <- e
			case error:
				log.Println(v)
				return
			}
		}
	}()
	return c
}

func pushAll(event RealtimeEvent, chans map[bson.ObjectId]chan RealtimeEvent) {
	for _, channel := range chans {
		channel <- event
	}
}

func (realtime *RealtimeRedis) GetReltChannel(id bson.ObjectId) ReltChannel {
	log.Println("getting realtime channel")
	c := ReltChannel{make(map[bson.ObjectId](chan RealtimeEvent)), realtime.getChannel(id)}
	go func() {
		for event := range c.events {
			go pushAll(event, c.chans)
		}
	}()
	return c
}

func (realtime *RealtimeRedis) GetWSChannel(id bson.ObjectId) ReltWSChannel {
	log.Println("getting websocket channel for", id.Hex())
	c := make(chan RealtimeEvent, RELT_WS_BUFF_SIZE)
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
	channel       chan RealtimeEvent
	subscriptions []bson.ObjectId
}

type ReltChannel struct {
	chans  map[bson.ObjectId](chan RealtimeEvent)
	events chan RealtimeEvent
}
