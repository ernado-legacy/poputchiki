package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-martini/martini"
	"io"
	"labix.org/v2/mgo/bson"
	"log"
	"mime/multipart"
	"net/http"
	"time"
)

type UserDB interface {
	Get(id bson.ObjectId) *User
	GetUsername(username string) *User
	// GetAll() []*User
	AddToFavorites(id bson.ObjectId, favId bson.ObjectId) error
	RemoveFromFavorites(id bson.ObjectId, favId bson.ObjectId) error
	Add(u *User) error
	GetFavorites(id bson.ObjectId) []*User
	Update(u *User) error
	// Delete(id bson.ObjectId) error
	AddGuest(id bson.ObjectId, guest bson.ObjectId) error
	GetAllGuests(id bson.ObjectId) ([]*User, error)
	AddMessage(m *Message) error
	GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) ([]*Message, error)
	GetMessage(id bson.ObjectId) (message *Message, err error)
	RemoveMessage(id bson.ObjectId) error
	AddToBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error
	RemoveFromBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error
}

type TokenStorage interface {
	Get(hexToken string) (*Token, error)
	Generate(user *User) (*Token, error)
	Remove(token *Token) error
}

type TokenInterface interface {
	Get() (*Token, error)
}

type RealtimeInterface interface {
	Push(id bson.ObjectId, event interface{}) error
	RealtimeHandler(w http.ResponseWriter, r *http.Request, token TokenInterface) (int, []byte)
}

const (
	JSON_HEADER     = "application/json; charset=utf-8"
	FORM_TARGET     = "target"
	FORM_EMAIL      = "email"
	FORM_PASSWORD   = "password"
	FORM_FIRSTNAME  = "firstname"
	FORM_SECONDNAME = "secondname"
	FORM_PHONE      = "phone"
	FORM_TEXT       = "text"
	FORM_FILE       = "file"
)

func JsonEncoder(c martini.Context, w http.ResponseWriter, r *http.Request) {
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

func GetUser(db UserDB, parms martini.Params, token TokenInterface) (int, []byte) {
	hexId := parms["id"]

	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()

	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	blacklisted := false
	for _, u := range user.Blacklist {
		if u == t.Id {
			blacklisted = true
		}
	}

	if blacklisted {
		return Render(ErrorBlacklisted)
	}

	// hiding private fields for non-owner
	if t == nil || t.Id != id {
		user.CleanPrivate()
	}

	return Render(user)
}

func AddToFavorites(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId = r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	favId := bson.ObjectIdHex(hexId)
	friend := db.Get(favId)
	if friend == nil {
		return Render(ErrorUserNotFound)
	}

	err := db.AddToFavorites(user.Id, friend.Id)
	if err != nil {
		return Render(ErrorBadRequest)
	}

	return Render("updated")
}

func AddToBlacklist(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId = r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	favId := bson.ObjectIdHex(hexId)
	friend := db.Get(favId)
	if friend == nil {
		return Render(ErrorUserNotFound)
	}

	err := db.AddToBlacklist(user.Id, friend.Id)
	if err != nil {
		return Render(ErrorBadRequest)
	}

	return Render("added to blacklist")
}

func RemoveFromBlacklist(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId = r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	favId := bson.ObjectIdHex(hexId)
	friend := db.Get(favId)
	if friend == nil {
		return Render(ErrorUserNotFound)
	}

	err := db.RemoveFromBlacklist(user.Id, friend.Id)
	if err != nil {
		return Render(ErrorBadRequest)
	}

	return Render("removed")
}

func RemoveFromFavorites(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId = r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	favId := bson.ObjectIdHex(hexId)
	friend := db.Get(favId)
	if friend == nil {
		return Render(ErrorUserNotFound)
	}

	err := db.RemoveFromFavorites(user.Id, friend.Id)
	if err != nil {
		return Render(ErrorBadRequest)
	}

	return Render("removed")
}

func GetFavorites(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	favorites := db.GetFavorites(id)
	if favorites == nil {
		return Render(ErrorUserNotFound)
	}

	for key, _ := range favorites {
		favorites[key].CleanPrivate()
	}

	return Render(favorites)
}

func GetGuests(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	guests, err := db.GetAllGuests(id)

	if err != nil {
		return Render(ErrorBackend)
	}

	if guests == nil {
		return Render(ErrorUserNotFound)
	}

	for key, _ := range guests {
		guests[key].CleanPrivate()
	}

	return Render(guests)
}

func AddToGuests(db UserDB, parms martini.Params, r *http.Request, token TokenInterface, realtime RealtimeInterface) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId = r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	guestId := bson.ObjectIdHex(hexId)
	guest := db.Get(guestId)
	if guest == nil {
		return Render(ErrorUserNotFound)
	}

	go func() {
		err := db.AddGuest(user.Id, guest.Id)
		if err != nil {
			log.Println(err)
		}
	}()

	return Render("added to guests")
}

func Login(db UserDB, r *http.Request, tokens TokenStorage) (int, []byte) {
	username, password := r.FormValue(FORM_EMAIL), r.FormValue(FORM_PASSWORD)
	user := db.GetUsername(username)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	if user.Password != getHash(password) {
		log.Printf("%s != %s", user.Password, getHash(password))
		return Render(ErrorAuth)
	}

	t, err := tokens.Generate(user)
	if err != nil {
		return Render(ErrorBackend)
	}

	return Render(t)
}

func Logout(db UserDB, r *http.Request, tokens TokenStorage, token TokenInterface) (int, []byte) {
	t, _ := token.Get()

	if t == nil {
		return Render(ErrorAuth)
	}

	err := tokens.Remove(t)

	if err != nil {
		return Render(ErrorBackend)
	}

	return Render("logged out")
}

func Register(db UserDB, r *http.Request, tokens TokenStorage) (int, []byte) {
	u := UserFromForm(r)
	uDb := db.GetUsername(u.Email)
	if uDb != nil {
		return Render(ErrorBadRequest) // todo: change error name
	}

	err := db.Add(u)

	if err != nil {
		log.Println(err)
		return Render(ErrorBadRequest) // todo: change error name
	}

	t, err := tokens.Generate(u)
	if err != nil {
		return Render(ErrorBackend)
	}

	return Render(t)
}

func Update(db UserDB, r *http.Request, token TokenInterface, parms martini.Params) (int, []byte) {
	hexId := parms["id"]
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	id := bson.ObjectIdHex(hexId)
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	UpdateUserFromForm(r, user)
	err := db.Update(user)

	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(user)
}

func SendMessage(db UserDB, parms martini.Params, r *http.Request, token TokenInterface, realtime RealtimeInterface) (int, []byte) {
	text := r.FormValue(FORM_TEXT)

	if text == BLANK {
		return Render(ErrorBadRequest)
	}

	destinationHex := parms["id"]

	if !bson.IsObjectIdHex(destinationHex) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	destination := bson.ObjectIdHex(destinationHex)
	origin := t.Id

	now := time.Now()
	m1 := Message{bson.NewObjectId(), origin, origin, destination, now, text}
	m2 := Message{bson.NewObjectId(), destination, origin, destination, now, text}

	go func() {

		u := db.Get(destination)
		blacklisted := false
		for _, id := range u.Blacklist {
			if id == origin {
				blacklisted = true
			}
		}

		if blacklisted {
			err := realtime.Push(origin, MessageSendBlacklisted{m1.Id})
			if err != nil {
				log.Println(err)
			}
		}

		err := realtime.Push(origin, m1)
		if err != nil {
			log.Println(err)
		}

		err = realtime.Push(destination, m2)
		if err != nil {
			log.Println(err)
		}

		err = db.AddMessage(&m1)
		if err != nil {
			log.Println(err)
			return
		}

		err = db.AddMessage(&m2)
		if err != nil {
			log.Println(err)
		}
	}()

	return Render("message sent")
}

func RemoveMessage(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	idHex := parms["id"]

	if !bson.IsObjectIdHex(idHex) {
		return Render(ErrorBadId)
	}

	id := bson.ObjectIdHex(idHex)

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	message, err := db.GetMessage(id)

	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	if message.User != t.Id {
		return Render(ErrorNotAllowed)
	}

	go func() {
		err := db.RemoveMessage(id)

		if err != nil {
			log.Println(err)
		}
	}()

	return Render("message removed")
}

func GetMessagesFromUser(db UserDB, parms martini.Params, r *http.Request, token TokenInterface) (int, []byte) {
	originHex := parms["id"]
	if !bson.IsObjectIdHex(originHex) {
		return Render(ErrorBadId)
	}

	t, _ := token.Get()
	if t == nil {
		return Render(ErrorAuth)
	}

	origin := bson.ObjectIdHex(originHex)
	destination := t.Id

	messages, err := db.GetMessagesFromUser(destination, origin)

	if err != nil {
		return Render(ErrorBackend)
	}

	if messages == nil {
		return Render(ErrorUserNotFound) // todo: rename error
	}

	return Render(messages)
}

func UploadImage(db UserDB, parms martini.Params, r *http.Request, token TokenInterface, realtime RealtimeInterface) (int, []byte) {
	t, _ := token.Get()
	client := &http.Client{}
	f, h, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read from file", err)
		return Render(ErrorBackend)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(FORM_FILE, h.Filename)
	length := r.ContentLength

	// 10 Mb
	if length > 1024*1024*100 {
		return Render(ErrorBadRequest)
	}

	var p float32
	var read int64
	bufLen := length / 50
	for {
		buffer := make([]byte, bufLen)
		cBytes, err := f.Read(buffer)
		if err == io.EOF {
			break
		}
		read = read + int64(cBytes)
		//fmt.Printf("read: %v \n",read )
		p = float32(read) / float32(length) * 100
		log.Printf("progress: %v \n", p)
		if t != nil {
			realtime.Push(t.Id, ProgressMessage{p})
		}

		part.Write(buffer[0:cBytes])
	}
	// _, err = io.Copy(part, f)
	err = writer.Close()
	if err != nil {
		return Render(ErrorBackend)
	}
	urlStr := "http://localhost:9333/dir/assign"
	req, err := http.NewRequest("GET", urlStr, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	rbody := &bytes.Buffer{}
	_, err = rbody.ReadFrom(resp.Body)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	resp.Body.Close()
	assign := WeedAssign{}
	err = json.Unmarshal(rbody.Bytes(), &assign)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	urlStr = "http://" + assign.Url + "/" + assign.Fid
	req, err = http.NewRequest("POST", urlStr, body)
	req.Header.Add("Content-type", writer.FormDataContentType())
	resp, err = client.Do(req)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	im := Image{bson.NewObjectId(), assign.Fid, urlStr}
	return Render(im)
}
