package main

import (
	"encoding/json"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"strconv"
)

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

func TokenWrapper(c martini.Context, r *http.Request, tokens TokenStorage, w http.ResponseWriter) {
	var hexToken string
	q := r.URL.Query()

	tStr := q.Get(TOKEN_URL_PARM)

	if tStr != BLANK {
		hexToken = tStr
	}

	tCookie, err := r.Cookie("token")

	if err == nil {
		hexToken = tCookie.Value
	}

	token, err := tokens.Get(hexToken)
	if err != nil {
		log.Println(err)
		code, data := Render(ErrorBackend)
		http.Error(w, string(data), code) // todo: set content-type
	}

	c.Map(token)
}

func IdEqualityRequired(w http.ResponseWriter, id bson.ObjectId, t *Token) {
	if t.Id != id {
		log.Println(t.Id.Hex(), id)
		code, data := Render(ErrorNotAllowed)
		http.Error(w, string(data), code) // todo: set content-type
		return
	}
}

func WebpWrapper(c martini.Context, r *http.Request) {
	var accept WebpAccept
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
	accept = WebpAccept(val == 1)
	c.Map(accept)
}

func JsonEncoderWrapper(r *http.Request, c martini.Context) {
	c.Map(json.NewDecoder(r.Body))
}

func IdWrapper(c martini.Context, r *http.Request, tokens TokenStorage, w http.ResponseWriter, parms martini.Params) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		code, data := Render(ErrorBadId)
		http.Error(w, string(data), code) // todo: set content-type
		return
	}
	c.Map(bson.ObjectIdHex(hexId))
}

func NeedAuth(res http.ResponseWriter, t *Token) {
	if t == nil {
		code, resp := Render(ErrorAuth)
		res.WriteHeader(code)
		res.Write(resp)
	}
}
