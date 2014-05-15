package main

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"labix.org/v2/mgo/bson"
	"reflect"
	"strings"
	"time"
)

const (
	REALTIME_REDIS_KEY   = "realtime"
	REALTIME_CHANNEL_KEY = "channel"
)

type RealtimeRedis struct {
	conn redis.Conn
}

func (realtime *RealtimeRedis) Push(id bson.ObjectId, event interface{}) error {
	t := strings.ToLower(reflect.TypeOf(event).Name())
	e := RealtimeEvent{t, event, time.Now()}
	args := []string{redisName, REALTIME_REDIS_KEY, REALTIME_CHANNEL_KEY, id.Hex()}
	key := strings.Join(args, REDIS_SEPARATOR)
	eJson, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = realtime.conn.Do("PUBLISH", key, eJson)
	return err
}
