package main

import (
	"encoding/json"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/models"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	QUERY_PAGINATION_COUNT  = "count"
	QUERY_PAGINATION_OFFSET = "offset"
)

type Redirect struct {
	Url string
}

func JsonEncoder(c martini.Context, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "" {
		log.Println("not setting header")
		return
	}

	w.Header().Set("Content-Type", JSON_HEADER)
}

func Render(value interface{}) (int, []byte) {
	// trying to marshal to json
	j, err := json.Marshal(value)
	if err != nil {
		j, err = json.Marshal(ErrorMarshal)
		if err != nil {
			log.Println(err)
			panic(err)
		}
		return ErrorMarshal.Code, j
	}
	switch v := value.(type) {
	case Error:
		if v.Code == http.StatusInternalServerError {
			log.Println(v)
		}
		return v.Code, j
	default:
		return http.StatusOK, j
	}
}

func SetOnlineWrapper(db models.DataBase, t *gotok.Token) {
	go db.SetOnline(t.Id)
	go db.SetLastActionNow(t.Id)
}

func PaginationWrapper(c martini.Context, r *http.Request) {
	q := r.URL.Query()
	p := models.Pagination{}

	if len(q[QUERY_PAGINATION_COUNT]) == 1 {
		p.Count, _ = strconv.Atoi(q[QUERY_PAGINATION_COUNT][0])
	}
	if len(q[QUERY_PAGINATION_OFFSET]) == 1 {
		p.Offset, _ = strconv.Atoi(q[QUERY_PAGINATION_OFFSET][0])
	}

	c.Map(p)
}

func ParserWrapper(c martini.Context, r *http.Request) {
	c.Map(models.NewParser(r))
}

func TokenWrapper(c martini.Context, r *http.Request, tokens gotok.Storage, w http.ResponseWriter) {
	var hexToken string
	q := r.URL.Query()

	tStr := q.Get(TOKEN_URL_PARM)

	if tStr != "" {
		hexToken = tStr
	}

	tCookie, err := r.Cookie("token")

	if err == nil {
		hexToken = tCookie.Value
	}
	tokenChannel := make(chan *gotok.Token)
	errorChannel := make(chan error)
	go func() {
		token, err := tokens.Get(hexToken)
		if err == nil {
			tokenChannel <- token
		} else {
			errorChannel <- err
		}
	}()

	select {
	case err := <-errorChannel:
		log.Println(err)
		code, data := Render(ErrorBackend)
		http.Error(w, string(data), code) // todo: set content-type
	case token := <-tokenChannel:
		c.Map(token)
	case <-time.After(time.Second * 5):
		log.Println("token system timed out")
		code, data := Render(ErrorBackend)
		http.Error(w, string(data), code) // todo: set content-type
	}
}

func IdEqualityRequired(w http.ResponseWriter, id bson.ObjectId, t *gotok.Token) {
	if t.Id != id {
		log.Println(t.Id.Hex(), id)
		code, data := Render(ErrorNotAllowed)
		http.Error(w, string(data), code) // todo: set content-type
		return
	}
}

func WebpWrapper(c martini.Context, r *http.Request) {
	var accept models.WebpAccept
	accept = false
	cookie, err := r.Cookie("webp")
	if err != nil {
		c.Map(accept)
		return
	}
	val, err := strconv.Atoi(cookie.Value)
	if err != nil {
		c.Map(accept)
		return
	}
	accept = models.WebpAccept(val == 1)
	c.Map(accept)
}

func VideoWrapper(c martini.Context, r *http.Request) {
	var accept models.VideoAccept = "none"
	cookie, err := r.Cookie("video")
	if err != nil {
		c.Map(accept)
		return
	}
	if cookie.Value != "" {
		accept = models.VideoAccept(cookie.Value)
	}
	c.Map(accept)
}

func AudioWrapper(c martini.Context, r *http.Request) {
	var accept models.AudioAccept = "none"
	cookie, err := r.Cookie("audio")
	if err != nil {
		c.Map(accept)
		return
	}
	if cookie.Value != "" {
		accept = models.AudioAccept(cookie.Value)
	}
	c.Map(accept)
}

func JsonEncoderWrapper(r *http.Request, c martini.Context) {
	c.Map(json.NewDecoder(r.Body))
}

func IdWrapper(c martini.Context, r *http.Request, tokens gotok.Storage, w http.ResponseWriter, parms martini.Params) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		code, data := Render(ErrorBadId)
		http.Error(w, string(data), code) // todo: set content-type
		return
	}
	c.Map(bson.ObjectIdHex(hexId))
}

func NeedAuth(res http.ResponseWriter, t *gotok.Token) {
	if t == nil {
		code, resp := Render(ErrorAuth)
		res.WriteHeader(code)
		res.Write(resp)
	}
}
