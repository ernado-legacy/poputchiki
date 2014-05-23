package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"labix.org/v2/mgo/bson"
	"strings"
	"time"
)

type TokenAbstract interface{}

const (
	TOKEN_REDIS_KEY = "tokens"
	TOKEN_URL_PARM  = "token"
	REDIS_SEPARATOR = ":"
)

func (u *User) GenerateToken() Token {
	hash := sha256.New()
	hash.Write([]byte(u.Email))
	hash.Write([]byte(u.Id))
	binary.Write(hash, binary.LittleEndian, time.Now().Unix())
	return Token{u.Id, hex.EncodeToString(hash.Sum(nil))}
}

type Token struct {
	Id    bson.ObjectId `json:"id"  bson:"_id,omitempty"`
	Token string        `json:"token"  bson:"token"`
}

type TokenStorageRedis struct {
	pool *redis.Pool
}

func (storage *TokenStorageRedis) Get(hexToken string) (*Token, error) {
	// checking token
	conn := storage.pool.Get()
	key := strings.Join([]string{redisName, TOKEN_REDIS_KEY, hexToken}, REDIS_SEPARATOR)
	reply, err := conn.Do("GET", key)

	if err != nil {
		return nil, err
	}

	if reply == nil {
		return nil, nil
	}

	// getting token from token storage
	t := Token{}
	err = json.Unmarshal(reply.([]byte), &t)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (storage *TokenStorageRedis) Generate(user *User) (*Token, error) {
	conn := storage.pool.Get()
	t := user.GenerateToken()
	tJson, err := json.Marshal(t)

	if err != nil {
		return nil, err
	}

	key := strings.Join([]string{redisName, TOKEN_REDIS_KEY, t.Token}, REDIS_SEPARATOR)
	_, err = conn.Do("SET", key, tJson)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (storage *TokenStorageRedis) Remove(token *Token) error {
	conn := storage.pool.Get()
	key := strings.Join([]string{redisName, TOKEN_REDIS_KEY, token.Token}, REDIS_SEPARATOR)
	_, err := conn.Do("DEL", key)

	return err
}

type TokenHanlder struct {
	e     error
	token *Token
}

func (t TokenHanlder) Get() (*Token, error) {
	if t.e != nil {
		return nil, t.e
	}

	return t.token, nil
}
