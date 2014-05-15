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
)

type RealtimeRedis struct {
	pool *redis.Pool
}

func (realtime *RealtimeRedis) Push(id bson.ObjectId, event interface{}) error {
	conn := realtime.pool.Get()
	defer conn.Close()
	t := strings.ToLower(reflect.TypeOf(event).Name())
	e := RealtimeEvent{t, event, time.Now()}
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, id.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	eJson, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = conn.Do("PUBLISH", key, eJson)
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
		log.Println(event)
		err := conn.WriteJSON(event)
		if err != nil {
			log.Println(err)
			return Render(ErrorBackend)
		}
	}
	return Render("ok")
}

func (realtime *RealtimeRedis) getChannel(id bson.ObjectId) chan RealtimeEvent {
	c := make(chan RealtimeEvent, 100)
	conn := realtime.pool.Get()
	psc := redis.PubSubConn{conn}
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, id.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	psc.Subscribe(key)
	go func() {
		log.Println("route started")
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				log.Printf("%s: message: %s\n", v.Channel, v.Data)
				e := RealtimeEvent{}
				err := json.Unmarshal(v.Data, &e)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println("sending")
				c <- e
			case error:
				log.Println(v)
				return
			}
		}
		// for {
		// 	reply, err := realtime.conn.Receive()
		// 	log.Println("recieved", string(reply.([]byte)), err)
		// 	if err != nil {
		// 		log.Println(err)
		// 		return
		// 	}
		// 	e := RealtimeEvent{}
		// 	err = json.Unmarshal(reply.([]byte), &e)
		// 	if err != nil {
		// 		log.Println(err)
		// 		return
		// 	}
		// 	log.Println("sending")
		// 	c <- e
	}()
	return c
}
