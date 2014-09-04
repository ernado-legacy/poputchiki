package activities

import (
	"errors"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/models"
	"github.com/go-martini/martini"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math"
	"time"
)

var (
	ActivityNotFound = errors.New("Activity not found")
)

const (
	Message = "activity:message"
	Invite  = "activity:invite"
	Photo   = "activity:photo"
	Like    = "activity:like"
	Status  = "activity:status"
	Promo   = "activity:promo"
	Video   = "activity:video"
)

const (
	degradation = 0.8
	percentile  = 0.9
)

type Engine struct {
	duration   time.Duration
	database   models.DataBase
	activities map[string]models.Activity
	total      float64
}

func (e *Engine) get(key string) *models.Activity {
	activity, ok := e.activities[key]
	if ok {
		return &activity
	}
	return nil
}

func (e *Engine) add(key string, weight float64, count int) {
	e.activities[key] = models.Activity{key, weight, count}
	e.total += weight
}

func (e *Engine) delta(activity *models.Activity, count int) float64 {
	weight := activity.Weight
	log.Println("weight", weight)
	first := percentile * weight * (degradation - 1) / (math.Pow(degradation, float64(activity.Count)) - 1)
	return first * math.Pow(degradation, float64(count-1))
}

func (e *Engine) process(user bson.ObjectId, activity *models.Activity) error {
	count, err := e.database.GetActivityCount(user, activity.Key, e.duration)
	if err != nil {
		return err
	}
	delta := e.delta(activity, count+1)
	log.Println("[activities]", user.Hex(), delta, count)
	if err := e.database.ChangeRating(user, delta); err != nil {
		return err
	}
	return e.database.AddActivity(user, activity.Key)
}

func (e *Engine) getActivityProcessor(key string) func(*gotok.Token) {
	activity := e.get(key)
	if activity == nil {
		log.Fatal(ActivityNotFound)
	}
	processor := func(token *gotok.Token) {
		go func() {
			log.Println("processing", activity.Key)
			if err := e.process(token.Id, activity); err != nil {
				log.Println("[activities]", err)
			}
		}()
	}
	return processor
}

type Handler interface {
	Handle(key string)
}

type activityHandler struct {
	t *gotok.Token
	e *Engine
}

func (h *activityHandler) Handle(key string) {
	h.e.getActivityProcessor(key)(h.t)
}

func (e *Engine) Wrapper(t *gotok.Token, c martini.Context) {
	if t != nil {
		c.MapTo(&activityHandler{t, e}, (*Handler)(nil))
	}
}

func New(db models.DataBase, duration time.Duration) *Engine {
	engine := new(Engine)
	engine.duration = duration
	engine.database = db
	engine.activities = make(map[string]models.Activity)
	engine.add(Promo, 100, 1)
	engine.add(Status, 100, 3)
	engine.add(Message, 50, 5)
	engine.add(Invite, 50, 2)
	engine.add(Photo, 40, 3)
	engine.add(Like, 30, 10)
	return engine
}
