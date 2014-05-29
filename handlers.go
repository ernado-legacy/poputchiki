package main

import (
	// "bytes"
	"encoding/json"
	"github.com/ginuerzh/weedo"
	// "github.com/go-martini/martini"
	// "io"
	"labix.org/v2/mgo/bson"
	"log"
	// "mime/multipart"
	"net/http"
	"time"
)

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

func ReadJson(r *http.Request, i *interface{}) {
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(i)
}

func GetUser(db UserDB, token TokenInterface, uid IdInterface) (int, []byte) {
	t := token.Get()
	id := uid.Get()
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

func AddToFavorites(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()

	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId := r.FormValue(FORM_TARGET)
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

func AddToBlacklist(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId := r.FormValue(FORM_TARGET)
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

func RemoveFromBlacklist(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId := r.FormValue(FORM_TARGET)
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

func RemoveFromFavorites(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId := r.FormValue(FORM_TARGET)
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

func GetFavorites(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()
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

func GetGuests(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()
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

func AddToGuests(db UserDB, uid IdInterface, r *http.Request, token TokenInterface, realtime RealtimeInterface) (int, []byte) {
	t := token.Get()
	id := uid.Get()
	if t.Id != id {
		return Render(ErrorNotAllowed)
	}

	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	hexId := r.FormValue(FORM_TARGET)
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
		return Render(ErrorAuth)
	}

	t, err := tokens.Generate(user)
	if err != nil {
		return Render(ErrorBackend)
	}

	return Render(t)
}

func Logout(db UserDB, r *http.Request, tokens TokenStorage, token TokenInterface) (int, []byte) {
	t := token.Get()
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

func Update(db UserDB, r *http.Request, token TokenInterface, uid IdInterface) (int, []byte) {
	id := uid.Get()
	t := token.Get()
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

func SendMessage(db UserDB, uid IdInterface, r *http.Request, token TokenInterface, realtime RealtimeInterface) (int, []byte) {
	text := r.FormValue(FORM_TEXT)

	if text == BLANK {
		return Render(ErrorBadRequest)
	}

	destination := uid.Get()
	t := token.Get()
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

func RemoveMessage(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	id := uid.Get()
	message, err := db.GetMessage(id)

	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	t := token.Get()
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

func GetMessagesFromUser(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	origin := uid.Get()
	t := token.Get()
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

func UploadImage(r *http.Request) (int, []byte) {
	c := weedo.NewClient("localhost", 9333)
	f, h, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read from file", err)
		return Render(ErrorBackend)
	}
	// body := &bytes.Buffer{}
	// writer := multipart.NewWriter(body)
	// part, err := writer.CreateFormFile(FORM_FILE, h.Filename)
	length := r.ContentLength

	if length > 1024*1024*20 {
		return Render(ErrorBadRequest)
	}

	// var p float32
	// var read int64
	// bufLen := length / 50
	// for {
	// 	buffer := make([]byte, bufLen)
	// 	cBytes, err := f.Read(buffer)
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	read = read + int64(cBytes)
	// 	//fmt.Printf("read: %v \n",read )
	// 	p = float32(read) / float32(length) * 100
	// 	if t != nil {
	// 		realtime.Push(t.Id, ProgressMessage{p})
	// 	}

	// 	part.Write(buffer[0:cBytes])
	// }

	// err = writer.Close()
	// if err != nil {
	// 	return Render(ErrorBackend)
	// }
	fid, size, err := c.AssignUpload(h.Filename, h.Header.Get("Content-Type"), f)
	log.Println(fid, size)

	if err != nil {
		return Render(ErrorBackend)
	}

	purl, url, err := c.GetUrl(fid)

	log.Println(purl, url)

	if err != nil {
		return Render(ErrorBackend)
	}

	im := Image{bson.NewObjectId(), fid, purl}
	return Render(im)
}

func AddStatus(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	status := &StatusUpdate{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(status)

	if err != nil {
		return Render(ErrorBadRequest)
	}

	t := token.Get()
	status, err = db.AddStatus(t.Id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func GetStatus(db UserDB, token TokenInterface, uid IdInterface) (int, []byte) {
	status, err := db.GetStatus(uid.Get())
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func RemoveStatus(db UserDB, token TokenInterface, uid IdInterface) (int, []byte) {
	err := db.RemoveStatusSecure(token.Get().Id, uid.Get())
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render("ok")
}

func GetCurrentStatus(db UserDB, token TokenInterface, uid IdInterface) (int, []byte) {
	status, err := db.GetCurrentStatus(uid.Get())
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func UpdateStatus(db UserDB, uid IdInterface, r *http.Request, token TokenInterface) (int, []byte) {
	status := &StatusUpdate{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(status)
	id := uid.Get()

	if err != nil {
		return Render(ErrorBadRequest)
	}

	t := token.Get()
	status, err = db.UpdateStatusSecure(t.Id, id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

// func GetStatuses(db UserDB, token TokenInterface, uid IdInterface) (int, []byte) {

// }
