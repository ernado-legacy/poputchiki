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
	REALTIME_BUFFER_SIZE = 100
)

type RealtimeRedis struct {
	pool *redis.Pool
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
	c := realtime.getChannel(id)
	for event := range c {
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
	c := make(chan RealtimeEvent, REALTIME_BUFFER_SIZE)

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
