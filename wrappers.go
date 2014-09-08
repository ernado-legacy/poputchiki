package main

import (
	"encoding/json"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki/models"
	"github.com/go-martini/martini"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
)

const (
	QUERY_PAGINATION_COUNT  = "count"
	QUERY_PAGINATION_OFFSET = "offset"
)

type Redirect struct {
	Url string
}

func JsonEncoder(c martini.Context, w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if strings.Index(accept, "json") == -1 {
		return
	}
	if r.Header.Get("Upgrade") != "" || r.Header.Get("X-Html") != "" {
		return
	}

	w.Header().Set("Content-Type", JSON_HEADER)
}

func addStatus(value interface{}, status int) []byte {
	j, err := json.Marshal(Response{status, value})
	if err != nil {
		log.Println(string(j), err)
		panic(err)
	}
	return j
}

type Response struct {
	Status   int         `json:"status"`
	Response interface{} `json:"response"`
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
			debug.PrintStack()
		}
		if *mobile {
			return http.StatusOK, addStatus(value, v.Code)
		}
		return v.Code, j
	default:
		if *mobile {
			return http.StatusOK, addStatus(value, 0)
		}
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
	token, err := tokens.Get(hexToken)
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	c.Map(token)
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
	if r.URL.Query().Get("webp") == "true" {
		accept = true
		c.Map(accept)
	}
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
	var accept models.VideoAccept = models.VaMp4
	urlAccept := r.URL.Query().Get("video")
	if urlAccept != "" {
		accept = models.VideoAccept(urlAccept)
		return
	}
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
	var accept models.AudioAccept = models.AaAac
	urlAccept := r.URL.Query().Get("audio")
	if urlAccept != "" {
		accept = models.AudioAccept(urlAccept)
		return
	}
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

func NeedAdmin(w http.ResponseWriter, isAdmin models.IsAdmin) {
	if !isAdmin {
		code, data := Render(ErrorAuth)
		http.Error(w, string(data), code)
		return
	}
}

func AdminWrapper(c martini.Context, w http.ResponseWriter, t *gotok.Token, db models.DataBase, r *http.Request, tokens gotok.Storage) {
	admin := false
	defer func() {
		c.Map(models.IsAdmin(admin))
	}()
	cookie, err := r.Cookie("admin")
	user := db.Get(t.Id)
	cookieExists := false
	if err == nil {
		cookieExists = true
	}
	if cookieExists {
		token, err := tokens.Get(cookie.Value)
		if err != nil {
			log.Println(err)
			return
		}
		if token != nil {
			newUser := db.Get(token.Id)
			if newUser.IsAdmin {
				user = newUser
			}
		}
	}
	if user == nil || !user.IsAdmin {
		return
	}
	admin = true
	if !cookieExists {
		http.SetCookie(w, &http.Cookie{Name: "admin", Value: t.Token, Path: "/"})
	}
}
