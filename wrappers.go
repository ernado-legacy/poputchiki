package main

import (
	"encoding/json"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
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
	var t TokenAbstract
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

	t = TokenHanlder{err, token}
	c.Map(t)
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
	idHandler := IdHandler{bson.ObjectIdHex(hexId)}
	c.Map(idHandler)
}

func NeedAuth(res http.ResponseWriter, token TokenInterface) {
	if token.Get() == nil {
		code, resp := Render(ErrorAuth)
		res.WriteHeader(code)
		res.Write(resp)
	}
}
