package main

import (
	"github.com/ernado/gorobokassa"
	"github.com/garyburd/redigo/redis"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
	"time"
)

const (
	TR_REDIS_KEY  = "transactions"
	TR_COLLECTION = TR_REDIS_KEY
)

// handle creation and closing transactions
type TransactionHandler struct {
	transactions *mgo.Collection
	pool         *redis.Pool
	client       *gorobokassa.Client
}

func NewTransactionHandler(pool *redis.Pool, db *mgo.Database, login, password1, password2 string) *TransactionHandler {
	h := &TransactionHandler{}
	h.pool = pool
	h.client = gorobokassa.New(login, password1, password2)
	h.transactions = db.C(TR_COLLECTION)
	return h
}

func (t *TransactionHandler) UpdateID() error {
	c := t.pool.Get()
	key := []string{redisName, TR_REDIS_KEY}
	var maxArray []int
	var max int
	err := t.transactions.Find(nil).Sort("-_id").Limit(1).Distinct("_id", &maxArray)
	if len(maxArray) == 1 {
		max = maxArray[0]
	}
	if err == mgo.ErrNotFound {
		max = 0
	} else if err != nil {
		return err
	}
	_, err = c.Do("SET", strings.Join(key, REDIS_SEPARATOR), max)
	return err
}

func (t *TransactionHandler) getID() (int, error) {
	c := t.pool.Get()
	key := []string{redisName, TR_REDIS_KEY}
	return redis.Int(c.Do("INCR", strings.Join(key, REDIS_SEPARATOR)))
}

func (t *TransactionHandler) getURL(transaction *Transaction) string {
	return t.client.URL(transaction.Id, transaction.Value, transaction.Description)
}

func (t *TransactionHandler) Start(id bson.ObjectId, value int, description string) (string, *Transaction, error) {
	tID, err := t.getID()
	if err != nil {
		return "", nil, err
	}
	transaction := &Transaction{tID, id, value, description, time.Now(), false}
	if err = t.transactions.Insert(transaction); err != nil {
		return "", nil, err
	}
	return t.getURL(transaction), transaction, nil
}

func (t *TransactionHandler) Close(r *http.Request) (*Transaction, error) {
	invoiceID, _, err := t.client.ResultInvoice(r)
	if err != nil {
		return nil, err
	}
	transaction := new(Transaction)
	selector := bson.M{"_id": invoiceID, "closed": false}
	err = t.transactions.Find(selector).One(transaction)
	if err != nil {
		return nil, err
	}
	err = t.transactions.Update(selector, bson.M{"$set": bson.M{"closed": true}})
	return transaction, err
}