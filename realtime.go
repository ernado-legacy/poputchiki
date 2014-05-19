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
	_, err = conn.Do("PUBLISH", key, eJson)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (realtime *RealtimeRedis) RealtimeHandler(w http.ResponseWriter, r *http.Request, token TokenInterface) (int, []byte) {
	t, _ := token.Get()

	if t == nil {
		return Render(ErrorAuth)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	id := bson.NewObjectId()
	c := realtime.GetWSChannel(id)
	defer realtime.CloseWs(c)

	for event := range c.channel {
		err := conn.WriteJSON(event)
		if err != nil {
			log.Println(err)
			return Render(ErrorBackend)
		}
	}
	return Render("ok")
}

func (realtime *RealtimeRedis) getChannel(id bson.ObjectId) chan RealtimeEvent {
	// creating new channel
	c := make(chan RealtimeEvent, RELT_BUFF_SIZE)

	conn := realtime.Conn()
	psc := redis.PubSubConn{conn}
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, id.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
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

func (realtime *RealtimeRedis) GetReltChannel(id bson.ObjectId) ReltChannel {
	c := ReltChannel{make(map[bson.ObjectId](chan RealtimeEvent)), realtime.getChannel(id)}
	c.events = realtime.getChannel(id)
	go func() {
		for event := range c.events {
			go func() {
				for _, channel := range c.chans {
					channel <- event
				}
			}()
		}
	}()
	return c
}

func (realtime *RealtimeRedis) GetWSChannel(id bson.ObjectId) ReltWSChannel {
	c := make(chan RealtimeEvent, RELT_WS_BUFF_SIZE)
	_, ok := realtime.chans[id]
	if !ok {
		realtime.chans[id] = realtime.GetReltChannel(id)
	}
	wsid := bson.NewObjectId()
	realtime.chans[id].chans[wsid] = c
	return ReltWSChannel{id: wsid, user: id, channel: c}
}

func (realtime *RealtimeRedis) CloseWs(c ReltWSChannel) {
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