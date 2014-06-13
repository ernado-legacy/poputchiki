package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
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
	hash.Write([]byte(bson.NewObjectId().Hex()))
	binary.Write(hash, binary.LittleEndian, time.Now().Unix())
	return Token{u.Id, hex.EncodeToString(hash.Sum(nil))}
}

type Token struct {
	Id    bson.ObjectId `json:"id"     bson:"user,omitempty"`
	Token string        `json:"token"  bson:"_id"`
}

type TokenStorageRedis struct {
	pool *redis.Pool
}

type TokenStorageMemory struct {
	tokens *mgo.Collection
	cache  map[string]bson.ObjectId
}

func (storage *TokenStorageMemory) Get(hexToken string) (*Token, error) {
	log.Println("getting token", hexToken)
	t := &Token{}
	value, ok := storage.cache[hexToken]
	if ok {
		t.Id = value
		t.Token = hexToken
		return t, nil
	}
	err := storage.tokens.Find(bson.M{"_id": hexToken}).One(t)
	if err == mgo.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	storage.cache[hexToken] = t.Id
	return t, nil
}

func (storage *TokenStorageMemory) Generate(user *User) (*Token, error) {
	t := user.GenerateToken()
	log.Println("maked token", t)
	err := storage.tokens.Insert(&t)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	storage.cache[t.Token] = t.Id
	return &t, nil
}

func (storage *TokenStorageMemory) Remove(token *Token) error {
	_, ok := storage.cache[token.Token]
	if ok {
		delete(storage.cache, token.Token)
	}
	return storage.tokens.Remove(bson.M{"_id": token.Token})
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

func (t TokenHanlder) Get() *Token {
	if t.e != nil {
		return nil
	}

	return t.token
}

type IdHandler struct {
	id bson.ObjectId
}

func (id IdHandler) Get() bson.ObjectId {
	return id.id
}
