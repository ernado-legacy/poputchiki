package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	conv "github.com/ernado/cymedia/mediad/models"
	"github.com/ernado/cymedia/mediad/query"
	"github.com/ernado/cymedia/photo"
	"github.com/ernado/gofbauth"
	"github.com/ernado/gosmsru"
	"github.com/ernado/gotok"
	"github.com/ernado/govkauth"
	"github.com/ernado/poputchiki/activities"
	. "github.com/ernado/poputchiki/models"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/rainycape/magick"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	THUMB_SIZE               = 200
	PHOTO_MAX_SIZE           = 1000
	PHOTO_MAX_MEGABYTES      = 20
	VIDEO_MAX_MEGABYTES      = 50
	VIDEO_MAX_LENGTH_SECONDS = 360
	VIDEO_BITRATE            = 500 * 1024
	AUDIO_BITRATE            = 128 * 1024
	VIDEO_SIZE               = 300
	JSON_HEADER              = "application/json; charset=utf-8"
	WEBP                     = "webp"
	JPEG                     = "jpeg"
	PNG                      = "png"
	WEBP_FORMAT              = "image/webp"
	JPEG_FORMAT              = "image/jpeg"
	FORM_TARGET              = "target"
	FORM_EMAIL               = "email"
	FORM_PASSWORD            = "password"
	FORM_FIRSTNAME           = "firstname"
	FORM_SECONDNAME          = "secondname"
	FORM_PHONE               = "phone"
	FORM_TEXT                = "text"
	FORM_FILE                = "file"
	FILTER_SEX               = "sex"
	FILTER_SEASON            = "season"
	FILTER_AGE_MIN           = "agemin"
	FILTER_AGE_MAX           = "agemax"
	FILTER_WEIGHT_MIN        = "weightmin"
	FILTER_WEIGHT_MAX        = "weightmax"
	FILTER_DESTINATION       = "destination"
	FILTER_GROWTH_MIN        = "growthmin"
	FILTER_GROWTH_MAX        = "growthmax"
)

var (
	ErrBadRequest     = errors.New("bad request") // internal bad request error
	ErrObjectNotFound = errors.New("Object not found")
)

// simple handler for testing the api from cyvisor
func Index() (int, []byte) {
	return Render("ok")
}

// GetUser handler for getting full user information
func GetUser(db DataBase, t *gotok.Token, id bson.ObjectId, context Context, u Updater) (int, []byte) {
	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// checking for blacklist
	for _, u := range user.Blacklist {
		if t != nil && u == t.Id {
			return Render(ErrorBlacklisted)
		}
	}
	if t == nil {
		user.CleanPrivate()
	}

	// hiding private fields for non-owner
	if t != nil && t.Id != id {
		user.CleanPrivate()
		go func() {
			defer func() {
				recover()
			}()
			origin := db.Get(t.Id)
			if origin.Invisible && origin.Vip {
				return
			}
			db.AddGuest(id, t.Id)
			u.Push(NewUpdate(id, t.Id, UpdateGuests, user))
		}()
	}
	return context.Render(user)
}

func GetCurrentUser(db DataBase, context Context, r Renderer) (int, []byte) {
	return r.Render(context.User)
}

type Target struct {
	Id bson.ObjectId `json:"target"`
}

// AddToFavorites adds target user to favorites of user
func AddToFavorites(db DataBase, id bson.ObjectId, r *http.Request, parser Parser) (int, []byte) {
	user := db.Get(id)
	// check user existance
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// get target user id from form
	target := &Target{}
	if err := parser.Parse(target); err != nil {
		return Render(ErrorBadId)
	}
	friend := db.Get(target.Id)
	// check for esistance
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	if user.Id == friend.Id {
		return Render(ValidationError(errors.New("Зачем вы таки пытаетесь добавить себя в подписчики?")))
	}
	// add to favorites
	if err := db.AddToFavorites(user.Id, friend.Id); err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("updated")
}

// AddToBlacklist adds target user to blacklist of user
func AddToBlacklist(db DataBase, id bson.ObjectId, r *http.Request, parser Parser) (int, []byte) {
	user := db.Get(id)
	// check existance
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// get target id from form data
	target := &Target{}
	if err := parser.Parse(target); err != nil {
		return Render(ErrorBadId)
	}
	friend := db.Get(target.Id)
	// check that target exists
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	if user.Id == friend.Id {
		return Render(ValidationError(errors.New("Зачем вы таки пытаетесь добавить себя в блеклист?")))
	}
	if err := db.AddToBlacklist(user.Id, friend.Id); err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("added to blacklist")
}

// RemoveFromBlacklist removes target user from blacklist of another user
func RemoveFromBlacklist(db DataBase, id bson.ObjectId, r *http.Request, parser Parser) (int, []byte) {
	user := db.Get(id)
	// check existance
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	target := &Target{}
	if err := parser.Parse(target); err != nil {
		return Render(ErrorBadId)
	}
	friend := db.Get(target.Id)
	// check that target exists
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	if err := db.RemoveFromBlacklist(user.Id, friend.Id); err != nil {
		if err == mgo.ErrNotFound {
			return Render(ErrorUserNotFound)
		}
		return Render(BackendError(err))
	}
	return Render("removed")
}

// RemoveFromFavorites removes target user from favorites of another user
func RemoveFromFavorites(db DataBase, id bson.ObjectId, r *http.Request, parser Parser) (int, []byte) {
	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	target := &Target{}
	if err := parser.Parse(target); err != nil {
		return Render(ErrorBadId)
	}
	friend := db.Get(target.Id)
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	if err := db.RemoveFromFavorites(user.Id, friend.Id); err != nil {
		if err == mgo.ErrNotFound {
			return Render(ErrorUserNotFound)
		}
		return Render(BackendError(err))
	}
	return Render("removed")
}

// GetFavorites returns list of users in favorites of target user
func GetFavorites(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	favorites := db.GetFavorites(id)
	// check for existance
	if favorites == nil {
		return Render([]interface{}{})
	}
	return context.Render(Users(favorites))
}

// GetFavorites returns list of users in favorites of target user
func GetFollowers(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	favorites, err := db.GetAllUsersWithFavorite(id)
	if err != nil && err != mgo.ErrNotFound {
		return Render(BackendError(err))
	}
	// check for existance
	if favorites == nil {
		return Render([]interface{}{})
	}
	return context.Render(Users(favorites))
}

// GetFavorites returns list of users in favorites of target user
func GetBlacklisted(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	blacklisted := db.GetBlacklisted(id)
	// check for existance
	if blacklisted == nil {
		return Render([]interface{}{})
	}
	return context.Render(Users(blacklisted))
}

// GetFavorites returns list of users in guests of target user
func GetGuests(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	guests, err := db.GetAllGuestUsers(id)
	if err != nil {
		return Render(BackendError(err))
	}
	// check for existance
	if guests == nil {
		return Render([]interface{}{})
	}
	for _, u := range guests {
		u.Prepare(context)
	}
	return context.Render(guests)
}

func AddToGuests(db DataBase, id bson.ObjectId, r *http.Request, realtime AutoUpdater) (int, []byte) {
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

	err := db.AddGuest(guest.Id, user.Id)
	if err != nil {
		log.Println("unable to add guest", err)
		return Render(ErrorBackend)
	}

	return Render("added to guests")
}

// Login checks the provided credentials and return token for user, setting appropriate
// auth cookies

type LoginCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(db DataBase, r *http.Request, w http.ResponseWriter, tokens gotok.Storage, parser Parser) (int, []byte) {
	credentials := new(LoginCredentials)
	if err := parser.Parse(credentials); err != nil {
		return Render(ValidationError(err))
	}
	username, password := strings.ToLower(credentials.Email), credentials.Password
	user := db.GetUsername(username)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	if user.Password != getHash(password, db.Salt()) {
		return Render(ErrorAuth)
	}
	t, err := tokens.Generate(user.Id)
	if err != nil {
		return Render(BackendError(err))
	}
	http.SetCookie(w, t.GetCookie())
	return Render(t)
}

// Logout ends the current session and makes current token unusable
func Logout(db DataBase, r *http.Request, tokens gotok.Storage, t *gotok.Token) (int, []byte) {
	if err := tokens.Remove(t); err != nil {
		return Render(BackendError(err))
	}
	return Render("logged out")
}

// Register checks the provided credentials, add new user with that credentials to
// database and returns new authorisation token, setting the appropriate cookies
func Register(db DataBase, r *http.Request, w http.ResponseWriter, tokens gotok.Storage, mail MailHtmlSender) (int, []byte) {
	// load user data from form
	u := UserFromForm(r, db.Salt())
	// check that email is unique
	uDb := db.GetUsername(u.Email)
	if uDb != nil {
		return Render(ErrorUserAlreadyRegistered)
	}
	// add to database
	u.Rating = 100.0
	u.Subscriptions = Subscriptions
	u.Balance = startCapital // TODO: Disable on production
	u.Registered = time.Now()
	if err := db.Add(u); err != nil {
		return Render(BackendError(err))
	}
	// generate token
	t, err := tokens.Generate(u.Id)
	if err != nil {
		return Render(BackendError(err))
	}
	// generate confirmation token for email confirmation
	confTok := db.NewConfirmationToken(u.Id)
	if confTok == nil {
		return Render(BackendError(errors.New("Unable to generate token")))
	}
	if !*development {
		type Data struct {
			Url   string
			Email string
		}
		data := Data{"http://poputchiki.ru/api/confirm/email/" + confTok.Token, u.Email}
		if err := mail.Send("registration.html", u.Id, "Подтверждение регистрации", data); err != nil {
			log.Println("[email]", err)
		}
	}
	http.SetCookie(w, t.GetCookie())
	return Render(t)
}

// Update updates user information with provided key-value document
func UpdateUser(db DataBase, id bson.ObjectId, parser Parser, context Context) (int, []byte) {
	user := new(User)
	query, err := parser.Query(user)

	if err != nil {
		log.Println("parser error", err)
		return Render(ValidationError(err))
	}

	isWriteable := func(key string) bool {
		for _, v := range UserWritableFields {
			if key == v {
				return true
			}
		}
		return false
	}

	// removing read-only fields
	for k := range query {
		if !isWriteable(k) {
			delete(query, k)
		}
	}
	// checking field count
	if len(query) == 0 {
		return context.Render(ValidationError(errors.New("no fields to change")))
	}
	// encoding to user - checking type & field existance
	user = new(User)
	err = convert(query, user)
	if err != nil {
		log.Println("converting error", err)
		return context.Render(ValidationError(err))
	}

	if user.Password != "" {
		user.Password = getHash(user.Password, context.DB.Salt())
	}

	// encoding back to query object
	// marshalling to bson
	newQuery := bson.M{}
	tmp, err := bson.Marshal(user)
	if err != nil {
		log.Println("marchaling error", err)
		return Render(ValidationError(err))
	}
	// unmarshalling to bson map
	err = bson.Unmarshal(tmp, &newQuery)
	if err != nil {
		log.Println("unmarchaling error", err)
		return context.Render(ValidationError(err))
	}

	// removing fields, that dont exist in initial query
	for k := range newQuery {
		_, ok := query[k]
		if !ok {
			delete(newQuery, k)
		}
	}
	// updating user
	_, err = context.DB.Update(id, newQuery)
	if err != nil {
		return context.Render(BackendError(err))
	}
	// returning updated user
	updated := context.DB.Get(id)
	return context.Render(updated)
}

func Must(err error) {
	if err != nil {
		log.Println(err)
	}
}

type MessageText struct {
	Text    string        `json:"text"`
	Origin  bson.ObjectId `json:"origin,omitempty"`
	ImageId bson.ObjectId `json:"image,omitempty"`
	Photo   string        `json:"photo"`
}

func SendMessage(context Context, db DataBase, parser Parser, destination bson.ObjectId, r *http.Request, t *gotok.Token, realtime AutoUpdater, admin IsAdmin) (int, []byte) {
	message := &MessageText{}
	err := parser.Parse(message)
	if err != nil {
		return Render(ValidationError(err))
	}

	text := message.Text
	origin := t.Id
	photo := message.Photo

	if len(message.Origin.Hex()) > 0 && admin {
		origin = message.Origin
		log.Println("administrative message forced origin", origin.Hex())
	}

	if text == "" && len(photo) == 0 {
		return Render(ValidationError(errors.New("Blank text provided")))
	}

	if len(photo) != 0 && bson.IsObjectIdHex(photo) {
		if !bson.IsObjectIdHex(photo) {
			return Render(ValidationError(errors.New("Invalid photo id")))
		}
		photoId := bson.ObjectIdHex(photo)
		p, err := db.GetPhoto(photoId)
		if err == mgo.ErrNotFound {
			Render(ValidationError(fmt.Errorf("Photo %s not found", photoId.Hex())))
		}
		if err != nil {
			Render(BackendError(err))
		}
		photo = p.ImageJpeg
	}

	m1, m2 := NewMessagePair(db, origin, destination, photo, text)
	if len(message.ImageId.Hex()) > 0 {
		p, err := db.GetPhoto(message.ImageId)
		if err != nil {
			errorText := fmt.Sprintf("Photo with id %s not found", message.ImageId.Hex())
			return Render(ValidationError(errors.New(errorText)))
		}
		m1.Photo = p.ImageJpeg
		m2.Photo = p.ImageJpeg
	}

	u := db.Get(destination)
	if u == nil {
		return Render(ErrorUserNotFound)
	}
	// check blacklist of destination
	for _, id := range u.Blacklist {
		if id == origin {
			return Render(ErrorBlacklisted)
		}
	}
	if err := realtime.Push(origin, m1); err != nil {
		Render(BackendError(err))
	}
	if err := realtime.Push(destination, m2); err != nil {
		Render(BackendError(err))
	}
	if err := db.AddMessage(m1); err != nil {
		Render(BackendError(err))
	}
	if err := db.AddMessage(m2); err != nil {
		Render(BackendError(err))
	}
	return context.Render(m1)
}

func SendInvite(db DataBase, parser Parser, engine activities.Handler, destination bson.ObjectId, r *http.Request, t *gotok.Token, realtime AutoUpdater) (int, []byte) {
	textDestination := "Вас пригласили в путешествие"
	textOrigin := "Вы отправили приглашение в путешествие"
	origin := t.Id
	toOrigin, toDestination := NewInvites(db, origin, destination, textOrigin, textDestination)
	u := db.Get(destination)
	if u == nil {
		return Render(ErrorUserNotFound)
	}
	// check blacklist of destination
	for _, id := range u.Blacklist {
		if id == origin {
			return Render(ErrorBlacklisted)
		}
	}
	// Must(realtime.Push(origin, toOrigin))
	Must(realtime.Push(destination, toDestination))

	Must(db.AddInvite(toOrigin))
	Must(db.AddInvite(toDestination))
	engine.Handle(activities.Invite)

	return Render("message sent")
}

func RemoveMessage(db DataBase, id bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
	message, err := db.GetMessage(id)
	if err != nil {
		return Render(BackendError(err))
	}
	if message.User != t.Id {
		return Render(ErrorNotAllowed)
	}
	go func() {
		Must(db.RemoveMessage(id))
	}()
	return Render("message removed")
}

func MarkReadMessage(db DataBase, id bson.ObjectId, t *gotok.Token) (int, []byte) {
	err := db.SetRead(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render("message marked as read")
}

func GetUnreadCount(db DataBase, t *gotok.Token) (int, []byte) {
	n, err := db.GetUnreadCount(t.Id)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render(UnreadCount{n})
}

func GetMessagesFromUser(origin bson.ObjectId, context Context, pagination Pagination, realtime RealtimeInterface) (int, []byte) {
	db := context.DB
	messages, err := db.GetMessagesFromUser(context.User.Id, origin, pagination)
	if err != nil && err != mgo.ErrNotFound {
		return Render(BackendError(err))
	}
	if messages == nil {
		return Render([]interface{}{})
	}
	if err := db.SetReadMessagesFromUser(context.User.Id, origin); err != nil {
		return Render(BackendError(err))
	}
	if err := sendCounters(db, context.Token, realtime); err != nil {
		return Render(BackendError(err))
	}
	return context.Render(messages)
}

func GetChat(db DataBase, pagination Pagination, context Context, parms martini.Params) (int, []byte) {
	if !bson.IsObjectIdHex(parms["user"]) {
		return Render(ErrorBadId)
	}
	if !bson.IsObjectIdHex(parms["chat"]) {
		return Render(ErrorBadId)
	}
	user, chat := bson.ObjectIdHex(parms["user"]), bson.ObjectIdHex(parms["chat"])
	messages, err := db.GetMessagesFromUser(user, chat, pagination)
	if err != nil {
		return Render(BackendError(err))
	}
	if messages == nil {
		return Render([]interface{}{})
	}
	if err = db.SetReadMessagesFromUser(user, chat); err != nil {
		log.Println("SetReadMessagesFromUser", err)
	}
	return context.Render(messages)
}

func RemoveChat(db DataBase, origin bson.ObjectId, t *gotok.Token) (int, []byte) {
	if err := db.RemoveChat(t.Id, origin); err != nil {
		if err == mgo.ErrNotFound {
			return Render(ErrorUserNotFound)
		}
		return Render(BackendError(err))
	}
	return Render("ok")
}

// GetChats returns all user chats
func GetChats(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	dialogs, err := db.GetChats(id)
	if err != nil {
		return Render(BackendError(err))
	}
	for k := range dialogs {
		if dialogs[k].User != nil {
			dialogs[k].User.Prepare(context)
			dialogs[k].User.CleanPrivate()
		}
		if dialogs[k].OriginUser != nil {
			dialogs[k].OriginUser.Prepare(context)
			dialogs[k].OriginUser.CleanPrivate()
		}
	}
	if len(dialogs) == 0 {
		return Render([]interface{}{})
	}
	return Render(dialogs)
}

// reads data from io.Reader, uploads it with type/format and returs fid, purl and error
func uploadToWeed(adapter StorageAdapter, reader io.Reader, t, format string) (string, string, int64, error) {
	fid, purl, size, err := adapter.Upload(reader, t, format)
	if err != nil {
		log.Println(t, format, err)
		return "", "", size, err
	}
	if err != nil {
		log.Println(err)
		return "", "", size, err
	}
	log.Println(t, format, "uploaded", purl)
	return fid, purl, size, nil
}

func uploadImageToWeed(adapter StorageAdapter, image *magick.Image, format string) (string, string, int64, error) {
	encodeReader, encodeWriter := io.Pipe()
	go func() {
		defer encodeWriter.Close()
		info := magick.NewInfo()
		info.SetFormat(format)
		if err := image.Encode(encodeWriter, info); err != nil {
			log.Println(err)
		}
	}()
	return uploadToWeed(adapter, encodeReader, "image", format)
}

func pushProgress(length int64, rate int64, progressWriter *io.PipeWriter, progressReader *io.PipeReader, realtime AutoUpdater, t *gotok.Token) {
	defer progressWriter.Close()
	var p float32
	var read int64
	bufLen := length / rate
	for {
		buffer := make([]byte, bufLen)
		cBytes, err := progressReader.Read(buffer)
		if err == io.EOF {
			break
		}
		read += int64(cBytes)
		//fmt.Printf("read: %v \n",read )
		p = float32(read) / float32(length) * 100
		if t != nil {
			realtime.Push(t.Id, ProgressMessage{t.Id, p})
		}
	}
}

func realtimeProgress(progress chan float32, realtime AutoUpdater, t *gotok.Token) {
	message := ProgressMessage{t.Id, 0.0}
	for currentProgress := range progress {
		message.Progress = currentProgress
		realtime.Push(t.Id, message)
	}
}

func convert(input interface{}, output interface{}) error {
	inputJson, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(inputJson, output)
}

func uploadPhoto(reader io.Reader, context Context) (*Photo, error) {
	uploader := photo.NewUploader(context.Storage, PHOTO_MAX_SIZE, THUMB_SIZE)
	result, err := uploader.Upload(reader)
	if err != nil {
		return nil, err
	}
	newPhoto, err := context.DB.AddPhoto(context.User.Id, result.Image(), result.Thumbnail())
	return newPhoto, err
}

func uploadPhotoHidden(context Context) (*Photo, error) {
	f, _, err := context.Request.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read form file", err)
		return nil, ErrBadRequest
	}

	length := context.Request.ContentLength
	if length > 1024*1024*PHOTO_MAX_MEGABYTES {
		return nil, ErrBadRequest
	}

	uploader := photo.NewUploader(context.Storage, PHOTO_MAX_SIZE, THUMB_SIZE)
	result, err := uploader.Upload(f)
	if err != nil {
		return nil, err
	}

	newPhoto, err := context.DB.AddPhotoHidden(context.User.Id, result.Image(), result.Thumbnail())
	return newPhoto, err
}

func UploadPhoto(r *http.Request, context Context, engine activities.Handler) (int, []byte) {
	f, _, err := context.Request.FormFile(FORM_FILE)
	if err != nil {
		return context.Render(ValidationError(err))
	}
	photo, err := uploadPhoto(f, context)
	if err == ErrBadRequest {
		return context.Render(ValidationError(err))
	}
	if err != nil {
		return context.Render(BackendError(err))
	}
	engine.Handle(activities.Photo)
	return context.Render(photo)
}

func UploadPhotoHidden(r *http.Request, context Context) (int, []byte) {
	photo, err := uploadPhotoHidden(context)
	if err == ErrBadRequest {
		return context.Render(ValidationError(err))
	}
	if err != nil {
		return context.Render(BackendError(err))
	}
	return context.Render(photo)
}

func AddStatus(db DataBase, r *http.Request, t *gotok.Token, parser Parser, engine activities.Handler, context Context) (int, []byte) {
	status := new(Status)
	if err := parser.Parse(status); err != nil {
		return Render(ValidationError(err))
	}

	if len(status.Text) == 0 {
		return Render(ValidationError(errors.New("Отправлен пустой статус")))
	}

	count, err := db.GetLastDayStatusesAmount(t.Id)
	if err != nil {
		return context.Render(BackendError(err))
	}

	allowed := statusesPerDay
	u := context.User
	if u.Vip {
		allowed = statusesPerDayVip
	}
	if count >= allowed {
		return context.Render(ErrorInsufficentFunds)
	}
	status, err = db.AddStatus(t.Id, status.Text)
	if err != nil {
		go db.IncBalance(t.Id, PromoCost)
		return context.Render(BackendError(err))
	}
	engine.Handle(activities.Status)
	return context.Render(status)
}

func GetStatus(db DataBase, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetStatus(id)
	if err == mgo.ErrNotFound {
		return Render("")
	}
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(status)
}

func RemoveStatus(db DataBase, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	if err := db.RemoveStatusSecure(t.Id, id); err != nil {
		return Render(ErrorBackend)
	}
	return Render("ok")
}

func GetCurrentStatus(db DataBase, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetCurrentStatus(id)
	if err == mgo.ErrNotFound {
		return Render("")
	}
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(status)
}

func UpdateStatus(db DataBase, id bson.ObjectId, r *http.Request, t *gotok.Token, parser Parser) (int, []byte) {
	status := &Status{}
	if err := parser.Parse(status); err != nil {
		return Render(ValidationError(err))
	}
	status, err := db.UpdateStatusSecure(t.Id, id, status.Text)
	if err != nil {
		return Render(BackendError(err))
	}

	return Render(status)
}

func addGeo(db DataBase, t *gotok.Token, r *http.Request) url.Values {
	q := r.URL.Query()
	u := db.Get(t.Id)
	if len(u.Location) == 2 && q.Get("location") == "" {
		q.Add(LocationArgument, fmt.Sprintf(LocationFormat, u.Location[0], u.Location[1]))
	}
	return q
}

func SearchPeople(db DataBase, pagination Pagination, r *http.Request, t *gotok.Token, context Context) (int, []byte) {
	q := addGeo(db, t, r)
	query, err := NewQuery(q)
	if err != nil {
		return context.Render(ValidationError(err))
	}
	if err := query.Validate(db); err != nil {
		return context.Render(ValidationError(err))
	}
	u := context.User
	if !u.Vip {
		query.Sponsor = ""
	}
	result, count, err := db.Search(query, pagination)
	if err != nil {
		return Render(BackendError(err))
	}
	Users(result).Prepare(context)
	return context.Render(SearchResult{result, count})
}

func GetUsersByEmail(db DataBase, t *gotok.Token, parm martini.Params, context Context) (int, []byte) {
	users, err := db.GetUsersByEmail(parm["email"])
	if err != nil {
		return context.Render(BackendError(err))
	}
	if len(users) == 0 {
		return context.Render([]interface{}{})
	}
	return context.Render(users)
}
func SearchStatuses(db DataBase, pagination Pagination, r *http.Request, t *gotok.Token, context Context) (int, []byte) {
	q := addGeo(db, t, r)
	query, err := NewQuery(q)
	if err != nil {
		return Render(ValidationError(err))
	}
	if err := query.Validate(db); err != nil {
		return Render(ValidationError(err))
	}
	u := context.User
	if !u.Vip {
		query.Sponsor = ""
	}
	result, err := db.SearchStatuses(query, pagination)
	if err != nil {
		return Render(BackendError(err))
	}
	for key, _ := range result {
		result[key].UserObject.Prepare(context)
	}

	return context.Render(result)
}

func SearchPhoto(db DataBase, pagination Pagination, r *http.Request, t *gotok.Token, context Context) (int, []byte) {
	q := addGeo(db, t, r)
	query, err := NewQuery(q)
	if err != nil {
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	if err := query.Validate(db); err != nil {
		return Render(ValidationError(err))
	}
	u := db.Get(t.Id)
	if !u.Vip {
		query.Sponsor = ""
	}
	result, err := db.SearchPhoto(query, pagination)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	for key, _ := range result {
		result[key].Prepare(context)
		if result[key].UserObject != nil {
			result[key].UserObject.Prepare(context)
		}
	}

	return context.Render(result)
}

func AllPhoto(db DataBase, paginaton Pagination, context Context) (int, []byte) {
	photo, count, err := db.SearchAllPhoto(paginaton)
	if err != nil {
		return context.Render(BackendError(err))
	}
	PhotoSlice(photo).Prepare(context)
	return context.Render(SearchResult{photo, count})
}

func GetUserPhoto(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	photo, err := db.GetUserPhoto(id)
	if err != nil {
		return Render(ErrorBackend)
	}

	for key := range photo {
		photo[key].Prepare(context)
	}

	return Render(photo)
}

type RandomCycle struct {
	pool *redis.Pool
	db   DataBase
}

const (
	redisCycle    = "promo-cycle"
	cycleDuration = time.Minute * 3
)

func (client *RandomCycle) Cycle() {
	log.Println("[random promo]", "started")
	defer func() {
		recover()
		log.Println("[random promo]", "finished")
	}()
	key := strings.Join([]string{redisName, redisCycle}, REDIS_SEPARATOR)
	type Data struct {
		Time time.Time `json:"time"`
	}
	var (
		last    time.Time
		fromNow time.Duration
	)
	now := time.Now()
	last = now.Add(-time.Hour)
	v := new(Data)
	conn := client.pool.Get()
	data, err := redis.Bytes(conn.Do("GET", key))
	if err != nil && err != redis.ErrNil {
		log.Println("[random promo]", err)
		conn.Close()
		return
	}
	fromNow = cycleDuration
	if err := json.Unmarshal(data, v); err == nil {
		last = v.Time
		fromNow = now.Sub(last)
		log.Println("[random promo]", "found last time", fromNow, "ago")
	}
	conn.Close()
	tick := func() error {
		now := time.Now()
		log.Println("[random promo]", "tick initiated")
		defer func() {
			log.Println("[random promo]", "processed", time.Now().Sub(now))
		}()
		conn = client.pool.Get()
		defer conn.Close()
		if err := RandomPromo(client.db); err != nil {
			log.Println("[random promo]", "database error")
			return err
		}
		last = now
		v.Time = last
		data, err = json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := conn.Do("SET", key, data); err != nil {
			return err
		}
		return nil
	}
	ticker := time.NewTicker(cycleDuration)
	if fromNow < cycleDuration {
		log.Println("[random promo]", "forcing sleep")
		time.Sleep(cycleDuration - fromNow)
	}
	if err := tick(); err != nil {
		log.Println("[random promo]", err)
		return
	}
	for _ = range ticker.C {
		if err := tick(); err != nil {
			log.Println("[random promo]", err)
			return
		}
	}
}

func RandomPromo(db DataBase) error {
	log.Println("[random promo]", "processing")
	user, err := db.RandomUser()
	if err != nil {
		return err
	}
	log.Println("[random promo] selectel user", user.Id.Hex(), user.Name)
	request := &StripeItemRequest{Type: "photo", Id: user.Avatar}
	s, err := addStripeItem(db, request, user)
	if err != nil {
		return err
	}
	log.Println("[random promo]", "added stripe item", s.Id.Hex())
	return nil
}

func addStripeItem(db DataBase, request *StripeItemRequest, user *User) (*StripeItem, error) {
	var media interface{}

	i := new(StripeItem)
	i.Id = bson.NewObjectId()
	i.User = user.Id
	i.Type = request.Type

	switch request.Type {
	case "video":
		video := db.GetVideo(request.Id)
		if video == nil {
			return nil, ErrObjectNotFound
		}
		i.ImageJpeg = video.ThumbnailJpeg
		i.ImageWebp = video.ThumbnailWebp
		media = video
	case "audio":
		audio := db.GetAudio(request.Id)
		if audio == nil {
			audio = db.GetAudio(user.Audio)
		}
		if audio == nil {
			return nil, ErrObjectNotFound
		}
		i.ImageJpeg = user.AvatarJpeg
		i.ImageWebp = user.AvatarWebp
		media = audio
	case "photo":
		p, err := db.GetPhoto(request.Id)
		if err != nil && err != mgo.ErrNotFound {
			return nil, err
		}
		if p == nil {
			return nil, ErrObjectNotFound
		}
		i.ImageJpeg = p.ThumbnailJpeg
		i.ImageWebp = p.ThumbnailWebp
		media = p
	default:
		return nil, ErrObjectNotFound
	}
	if media == nil {
		return nil, ErrObjectNotFound
	}
	return db.AddStripeItem(i, media)
}

func AddStripeItem(engine activities.Handler, db DataBase, t *gotok.Token, parser Parser, context Context) (int, []byte) {
	request := new(StripeItemRequest)
	if err := parser.Parse(request); err != nil {
		return Render(ValidationError(err))
	}
	if !*development {
		if db.DecBalance(context.Token.Id, PromoCost) != nil {
			return Render(ErrorInsufficentFunds)
		}
	}
	s, err := addStripeItem(db, request, context.User)
	if err != nil {
		db.IncBalance(context.Token.Id, PromoCost)
	}
	if err == ErrObjectNotFound {
		return Render(ErrorObjectNotFound)
	}
	if err != nil {
		return Render(BackendError(err))
	}
	engine.Handle(activities.Promo)
	return context.Render(s)
}

func GetStripe(db DataBase, pagination Pagination, context Context) (int, []byte) {
	stripe, err := db.GetStripe(pagination.Count, pagination.Offset)
	if err != nil {
		return Render(BackendError(err))
	}
	for _, v := range stripe {
		if err := v.Prepare(context); err != nil {
			log.Println(err)
			// return Render(ErrorBackend)
		}
	}
	return context.Render(stripe)
}

func EnableVip(db DataBase, t *gotok.Token, parm martini.Params) (int, []byte) {
	var months, days, price int
	duration := parm["duration"]
	if duration == "week" {
		days = 7
		price = vipWeek
	}
	if duration == "month" {
		months = 1
		price = vipMonth
	}
	if price == 0 {
		return Render(ValidationError(errors.New("Duration must me month or week")))
	}

	if !*development {
		err := db.DecBalance(t.Id, uint(price))
		if err != nil {
			return Render(ErrorInsufficentFunds)
		}
	}

	user := db.Get(t.Id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	now := time.Now()
	if now.After(user.VipTill) {
		user.VipTill = now
	}
	till := user.VipTill.AddDate(0, months, days)
	if err := db.SetVipTill(t.Id, till); err != nil {
		return Render(BackendError(err))
	}
	if err := db.SetVip(t.Id, true); err != nil {
		return Render(BackendError(err))
	}
	return Render("ok")
}

// GetToken returns active token body
func GetToken(t *gotok.Token) (int, []byte) {
	return Render(t)
}

// ConfirmEmail verifies and deletes confirmation token, sets confirmation flag to user
func ConfirmEmail(db DataBase, args martini.Params, w http.ResponseWriter, tokens gotok.Storage, r *http.Request) (int, []byte) {
	token := args["token"]
	if token == "" {
		return Render(ErrorBadRequest)
	}
	tok := db.GetConfirmationToken(token)
	if tok == nil {
		return Render(ValidationError(errors.New("Ссылка устарела или недействительна")))
	}
	userToken, err := tokens.Generate(tok.User)
	if err != nil {
		return Render(BackendError(err))
	}
	err = db.ConfirmEmail(userToken.Id)
	if err != nil {
		log.Println(err)
		return Render(BackendError(err))
	}
	http.SetCookie(w, userToken.GetCookie())
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return Render("email подтвержден")
}

func ConfirmPhone(db DataBase, args martini.Params, w http.ResponseWriter, tokens gotok.Storage) (int, []byte) {
	codeStr := args["token"]
	if codeStr == "" {
		return Render(ErrorBadRequest)
	}
	h := sha256.New()
	h.Sum([]byte(codeStr))
	h.Sum([]byte(salt))
	val := hex.EncodeToString(h.Sum(nil))
	tok := db.GetConfirmationToken(val)
	if tok == nil {
		return Render(ValidationError(errors.New("blank token")))
	}
	userToken, err := tokens.Generate(tok.User)
	if err != nil {
		return Render(BackendError(err))
	}
	err = db.ConfirmPhone(userToken.Id)
	if err != nil {
		return Render(BackendError(err))
	}
	http.SetCookie(w, userToken.GetCookie())
	return Render("телефон подтвержден")
}

func ConfirmPhoneStart(db DataBase, t *gotok.Token) (int, []byte) {
	code := rand.Intn(999)
	codeStr := fmt.Sprintf("%03d", code)
	h := sha256.New()
	h.Sum([]byte(codeStr))
	h.Sum([]byte(salt))
	token := hex.EncodeToString(h.Sum(nil))
	tok := db.NewConfirmationTokenValue(t.Id, token)
	if tok == nil {
		return Render(BackendError(errors.New("Unable to generate confirmation token")))
	}
	u := db.Get(t.Id)
	if u == nil {
		return Render(ErrorBackend)
	}
	client := gosmsru.Client{}
	client.Key = smsKey
	phone := u.Phone
	if phone == "" {
		return Render(ValidationError(errors.New("Blank phone")))
	}
	err := client.Send(phone, codeStr)
	if err != nil {
		return Render(ValidationError(err))
	}
	return Render("ok")
}

func GetTransactionUrl(db DataBase, args martini.Params, t *gotok.Token, handler *TransactionHandler, r *http.Request, w http.ResponseWriter) {
	valuePopiki, err := strconv.Atoi(args["value"])
	if err != nil || valuePopiki <= 0 {
		code, data := Render(ErrorBadRequest)
		http.Error(w, string(data), code)
		return
	}

	value, ok := popiki[valuePopiki]
	if !ok {
		err := errors.New(fmt.Sprintf("Неверное количество попиков: %d", valuePopiki))
		code, data := Render(ValidationError(err))
		http.Error(w, string(data), code)
		return
	}
	url, transaction, err := handler.Start(t.Id, value, robokassaDescription)
	if err != nil {
		log.Println(url, transaction, err)
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func RobokassaSuccessHandler(db DataBase, r *http.Request, handler *TransactionHandler) (int, []byte) {
	log.Println("processing robokassa", r.URL.String())
	transaction, err := handler.Close(r)
	if err != nil {
		log.Println("[CRITICAL ERROR]", "transaction", transaction)
		return Render(ValidationError(err))
	}

	value := transaction.Value
	var valuePopiki int
	for k, v := range popiki {
		if value == v {
			valuePopiki = k
		}
	}

	if valuePopiki == 0 {
		err := errors.New(fmt.Sprintf("Неверное количество попиков: %d", valuePopiki))
		log.Println("[CRITICAL ERROR]", "transaction", transaction, err)
		return Render(ValidationError(err))
	}

	err = db.IncBalance(transaction.User, uint(valuePopiki))
	if err != nil {
		log.Println("[CRITICAL ERROR]", "transaction", transaction)
		return Render(ErrorBadRequest)
	}
	return Render(transaction)
}

func TopUp(db DataBase, parser Parser, handler *TransactionHandler, t *gotok.Token) (int, []byte) {
	type invoice struct {
		Amount  int    `json:"amount"`
		Recepit string `json:"receipt"`
	}
	data := new(invoice)
	if err := parser.Parse(data); err != nil {
		return Render(ValidationError(err))
	}
	_, transaction, err := handler.Start(t.Id, data.Amount, robokassaDescription)
	if err != nil {
		return Render(BackendError(err))
	}
	if err := handler.End(transaction); err != nil {
		return Render(BackendError(err))
	}
	if err := db.IncBalance(t.Id, uint(transaction.Value)); err != nil {
		log.Println("[CRITICAL ERROR]", "transaction", transaction)
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	return Render(transaction)
}

func LikeVideo(t *gotok.Token, id bson.ObjectId, db DataBase, engine activities.Handler, u Updater) (int, []byte) {
	err := db.AddLikeVideo(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	engine.Handle(activities.Like)
	v := db.GetVideo(id)
	if v.User != t.Id {
		go u.Push(NewUpdate(v.User, t.Id, UpdateLikes, v))
		go func() {
			db.AddGuest(t.Id, v.User)
		}()
	}
	return Render(v)
}

func RestoreLikeVideo(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveLikeVideo(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render(db.GetVideo(id))
}

func GetLikersVideo(id bson.ObjectId, db DataBase, context Context) (int, []byte) {
	likers := db.GetLikesVideo(id)
	// check for existance
	if likers == nil {
		return Render([]interface{}{})
	}
	return context.Render(Users(likers))
}

func LikePhoto(t *gotok.Token, id bson.ObjectId, db DataBase, engine activities.Handler, u Updater) (int, []byte) {
	err := db.AddLikePhoto(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	engine.Handle(activities.Like)
	p, _ := db.GetPhoto(id)
	if p.User != t.Id {
		go u.Push(NewUpdate(p.User, t.Id, UpdateLikes, p))
		go func() {
			db.AddGuest(t.Id, p.User)
		}()
	}
	return Render(p)
}

func RestoreLikePhoto(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveLikePhoto(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	p, _ := db.GetPhoto(id)
	return Render(p)
}

func RemovePhoto(t *gotok.Token, id bson.ObjectId, db DataBase, admin IsAdmin) (int, []byte) {
	userId := t.Id
	if admin {
		p, err := db.GetPhoto(id)
		if err != nil {
			return Render(BackendError(err))
		}
		userId = p.User
	}
	err := db.RemovePhoto(userId, id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	if err != nil {
		return Render(BackendError(err))
	}
	err = db.AvatarRemove(userId, id)
	if err != nil && err != mgo.ErrNotFound {
		return Render(BackendError(err))
	}
	return Render("ok")
}

func RemoveVideo(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveVideo(t.Id, id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	if err != nil {
		return Render(BackendError(err))
	}
	return Render("ok")
}

func RemoveAudio(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveAudioSecure(t.Id, id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	if err != nil {
		return Render(BackendError(err))
	}
	return Render("ok")
}

func GetLikersPhoto(id bson.ObjectId, db DataBase, context Context) (int, []byte) {
	likers := db.GetLikesPhoto(id)
	// check for existance
	if likers == nil {
		return Render([]interface{}{})
	}
	return context.Render(Users(likers))
}

func LikeStatus(t *gotok.Token, id bson.ObjectId, db DataBase, u Updater) (int, []byte) {
	err := db.AddLikeStatus(t.Id, id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	if err != nil {
		return Render(BackendError(err))
	}
	s, _ := db.GetStatus(id)
	if s.User != t.Id {
		u.Push(NewUpdate(s.User, t.Id, UpdateLikes, s))
		go func() {
			db.AddGuest(t.Id, s.User)
		}()
	}
	return Render(s)
}

func RestoreLikeStatus(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveLikeStatus(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	s, _ := db.GetStatus(id)
	return Render(s)
}

func GetLikersStatus(id bson.ObjectId, db DataBase, context Context) (int, []byte) {
	likers := db.GetLikesStatus(id)
	// check for existance
	if likers == nil {
		return Render([]interface{}{})
	}
	return context.Render(Users(likers))
}

func GetCountries(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	coutries, err := db.GetCountries(start)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render(coutries)
}

func GetCities(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	country := req.URL.Query().Get("country")
	if country == "" {
		return Render(ValidationError(errors.New("Blank country")))
	}
	if !db.CountryExists(country) {
		return Render(ValidationError(errors.New("Country does not exist")))
	}
	cities, err := db.GetCities(start, country)
	if err != nil {
		return Render(BackendError(err))
	}
	if len(cities) > 100 {
		cities = cities[:100]
	}
	return Render(cities)
}

func GetPlaces(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	places, err := db.GetPlaces(start)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render(places)
}

func GetCityPairs(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	cities, err := db.GetCityPairs(start)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render(cities)
}

func ForgotPassword(db DataBase, args martini.Params, mail MailHtmlSender, context Context) (int, []byte) {
	email := args["email"]
	u := db.GetUsername(email)
	if u == nil {
		return Render(ErrorUserNotFound)
	}
	confTok := db.NewConfirmationToken(u.Id)
	if confTok == nil {
		return Render(BackendError(errors.New("Unable to generate token")))
	}
	if !*development {
		type Data struct {
			Url  string
			User *User
		}
		u.Prepare(context)
		data := Data{"http://poputchiki.ru/api/auth/reset/" + confTok.Token, u}
		if err := mail.Send("password.html", u.Id, "Восстановление пароля", data); err != nil {
			log.Println("[email]", err)
		}
	}
	return Render("ok")
}

func ResetPassword(db DataBase, r *http.Request, w http.ResponseWriter, args martini.Params, tokens gotok.Storage) {
	token := args["token"]
	if token == "" {
		code, data := Render(ValidationError(errors.New("Blank token")))
		http.Error(w, string(data), code) // todo: set content-type
	}
	tok := db.GetConfirmationToken(token)
	if tok == nil {
		code, data := Render(ValidationError(errors.New("Bad token")))
		http.Error(w, string(data), code) // todo: set content-type
	}
	userToken, err := tokens.Generate(tok.User)
	if err != nil {
		code, data := Render(BackendError(errors.New("Token generation error")))
		http.Error(w, string(data), code) // todo: set content-type
	}
	err = db.ConfirmEmail(userToken.Id)
	if err != nil {
		log.Println(err)
		code, data := Render(BackendError(errors.New("Token confirmation error")))
		http.Error(w, string(data), code) // todo: set content-type
	}
	http.SetCookie(w, userToken.GetCookie())
	http.Redirect(w, r, "/settings/password", http.StatusTemporaryRedirect)
}

func VkontakteAuthStart(r *http.Request, w http.ResponseWriter, client *govkauth.Client) {
	url := client.DialogURL()
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
}

func FacebookAuthStart(r *http.Request, w http.ResponseWriter, client *gofbauth.Client) {
	url := client.DialogURL()
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
}

func ExportPhoto(context Context, url string) *Photo {
	res, err := http.Get(url)
	if err != nil {
		return nil
	}
	p, err := uploadPhoto(res.Body, context)
	if err != nil {
		log.Println(err)
	}
	return p
}

func ExportThumbnail(adapter StorageAdapter, fid string) (thumbnail string, err error) {
	url, err := adapter.GetUrl(fid)
	if err != nil {
		return
	}
	res, err := http.Get(url)
	if err != nil {
		return
	}

	uploader := photo.NewUploader(adapter, PHOTO_MAX_SIZE, THUMB_SIZE)
	p, err := uploader.Upload(res.Body)
	if err != nil {
		return
	}
	return p.Thumbnail(), nil
}

func VkontakteAuthRedirect(context Context, db DataBase, r *http.Request, w http.ResponseWriter, adapter StorageAdapter, tokens gotok.Storage, client *govkauth.Client) {
	token, err := client.GetAccessToken(r)
	if err != nil {
		code, _ := Render(ErrorBadRequest)
		http.Error(w, "Авторизация невозможна", code)
		return
	}
	u := db.GetUsername(token.Email)
	if u == nil {
		newUser := &User{}
		newUser.Email = token.Email
		newUser.Password = "oauth"
		newUser.EmailConfirmed = true
		user, err := client.GetName(token.UserID)
		newUser.Name = user.Name
		newUser.Birthday = user.Birthday
		newUser.Rating = 100.0
		newUser.Sex = user.Sex
		newUser.Subscriptions = Subscriptions
		newUser.Balance = startCapital
		newUser.Registered = time.Now()
		err = db.Add(newUser)
		if err != nil {
			code, _ := Render(ErrorBackend)
			http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
			return
		}
		u = db.GetUsername(token.Email)

		if user.Photo != "" {
			p := ExportPhoto(context, user.Photo)
			if p != nil {
				db.SetAvatar(u.Id, p.Id)
			} else {
				log.Println("unable to set avatar")
			}
		}
	}
	userToken, err := tokens.Generate(u.Id)
	if err != nil {
		code, _ := Render(ErrorBackend)
		http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
		return
	}

	http.SetCookie(w, userToken.GetCookie())
	http.SetCookie(w, &http.Cookie{Name: "userId", Value: u.Id.Hex(), Path: "/"})
	if *mobile {
		_, data := Render(userToken)
		fmt.Fprint(w, string(data))
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func FacebookAuthRedirect(context Context, db DataBase, r *http.Request, adapter StorageAdapter, w http.ResponseWriter, tokens gotok.Storage, client *gofbauth.Client) {
	token, err := client.GetAccessToken(r)
	if err != nil {
		code, _ := Render(ErrorBadRequest)
		http.Error(w, "Авторизация невозможна", code)
		return
	}
	fbUser, err := client.GetUser(token.AccessToken)
	if err != nil {
		code, _ := Render(ErrorBadRequest)
		http.Error(w, "Ошибка авторизации", code)
		return
	}
	u := db.GetUsername(fbUser.Email)
	if u == nil {
		newUser := &User{}
		newUser.Email = fbUser.Email
		newUser.Password = "oauth"
		newUser.EmailConfirmed = true
		newUser.Name = fbUser.Name
		newUser.Balance = startCapital
		newUser.Birthday = fbUser.Birthday
		newUser.Rating = 100.0
		newUser.Registered = time.Now()
		newUser.Subscriptions = Subscriptions
		err = db.Add(newUser)
		if err != nil {
			code, _ := Render(ErrorBackend)
			http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
			return
		}
		u = db.GetUsername(fbUser.Email)
		if fbUser.Photo != "" {
			p := ExportPhoto(context, fbUser.Photo)
			if p != nil {
				db.SetAvatar(u.Id, p.Id)
			} else {
				log.Println("unable to set avatar")
			}
		}
	}
	userToken, err := tokens.Generate(u.Id)
	if err != nil {
		code, _ := Render(ErrorBackend)
		http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
		return
	}

	http.SetCookie(w, userToken.GetCookie())
	http.SetCookie(w, &http.Cookie{Name: "userId", Value: u.Id.Hex(), Path: "/"})
	if *mobile {
		_, data := Render(userToken)
		fmt.Fprint(w, string(data))
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func AdminView(w http.ResponseWriter, t *gotok.Token, db DataBase, r *http.Request, tokens gotok.Storage) {
	view, err := template.ParseFiles("static/html/index.html")
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	view.Execute(w, nil)
}

func PhotoView(w http.ResponseWriter, t *gotok.Token, db DataBase, r *http.Request, tokens gotok.Storage) {
	view, err := template.ParseFiles("static/html/photo.html")
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	view.Execute(w, nil)
}

func AdminMessages(w http.ResponseWriter) (int, []byte) {
	data, err := AllTemplates.Bytes("messages.html")
	if err != nil {
		return Render(BackendError(err))
	}
	return http.StatusOK, data
}

func AdminLogin(id bson.ObjectId, t *gotok.Token, db DataBase, w http.ResponseWriter, r *http.Request, tokens gotok.Storage) {
	userToken, err := tokens.Generate(id)
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	http.SetCookie(w, userToken.GetCookie())
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func UploadVideoFile(r *http.Request, client query.QueryClient, db DataBase, adapter StorageAdapter, t *gotok.Token) (int, []byte) {
	id := bson.NewObjectId()
	video := &Video{Id: id, User: t.Id, Time: time.Now()}
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		return Render(BackendError(err))
	}
	_, err = db.AddVideo(video)
	if err != nil {
		return Render(BackendError(err))
	}

	optsMpeg := new(conv.VideoOptions)
	optsMpeg.Audio.Format = "aac"
	optsMpeg.Video.Format = "h264"
	optsMpeg.Audio.Bitrate = AUDIO_BITRATE
	optsMpeg.Video.Bitrate = VIDEO_BITRATE
	optsMpeg.Video.Square = true
	optsMpeg.Duration = 15
	optsMpeg.Video.Height = VIDEO_SIZE
	optsMpeg.Video.Width = VIDEO_SIZE

	optsWebm := new(conv.VideoOptions)
	optsWebm.Video.Format = "libvpx"
	optsWebm.Audio.Format = "libvorbis"
	optsWebm.Audio.Bitrate = AUDIO_BITRATE
	optsWebm.Video.Bitrate = VIDEO_BITRATE
	optsWebm.Video.Square = true
	optsWebm.Duration = 15
	optsWebm.Video.Height = VIDEO_SIZE
	optsWebm.Video.Width = VIDEO_SIZE
	optsThmb := new(conv.ThumbnailOptions)
	optsThmb.Format = "png"
	fid, _, _, err := adapter.Upload(f, "video", "video")
	if err != nil {
		return Render(BackendError(err))
	}
	if err := client.Push(id.Hex(), fid, conv.ThumbnailType, optsThmb); err != nil {
		return Render(BackendError(err))
	}
	if err := client.Push(id.Hex(), fid, conv.VideoType, optsWebm); err != nil {
		return Render(BackendError(err))
	}
	if err := client.Push(id.Hex(), fid, conv.VideoType, optsMpeg); err != nil {
		return Render(BackendError(err))
	}
	return Render(video)
}

func GetUserVideo(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	v, err := db.GetUserVideo(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	return context.Render(VideoSlice(v))
}

func GetUserMedia(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	v, err := db.GetUserVideo(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	VideoSlice(v).Prepare(context)
	p, err := db.GetUserPhoto(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	PhotoSlice(p).Prepare(context)
	return context.Render(MakeMediaSlice(p, v))
}

func GetVideo(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	v := db.GetVideo(id)
	if v == nil {
		return Render(ErrorObjectNotFound)
	}
	return context.Render(v)
}

func GetPhoto(db DataBase, id bson.ObjectId, context Context) (int, []byte) {
	p, err := db.GetPhoto(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	return context.Render(p)
}

func UploadAudio(r *http.Request, client query.QueryClient, db DataBase, adapter StorageAdapter, t *gotok.Token) (int, []byte) {
	id := bson.NewObjectId()
	audio := &Audio{Id: id, User: t.Id, Time: time.Now()}

	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		return Render(ValidationError(err))
	}
	_, err = db.AddAudio(audio)
	if err != nil {
		return Render(BackendError(err))
	}
	optsAac := &conv.AudioOptions{Bitrate: AUDIO_BITRATE, Format: "mp3"}
	optsAac.Duration = 15
	optsVorbis := &conv.AudioOptions{Bitrate: AUDIO_BITRATE, Format: "libvorbis"}
	optsVorbis.Duration = 15
	fid, _, _, err := adapter.Upload(f, "audio", "audio")
	if err != nil {
		return Render(BackendError(err))
	}
	if err := client.Push(id.Hex(), fid, conv.AudioType, optsAac); err != nil {
		return Render(BackendError(err))
	}
	if err := client.Push(id.Hex(), fid, conv.AudioType, optsVorbis); err != nil {
		return Render(BackendError(err))
	}
	return Render(audio)
}

func GetCounters(db DataBase, t *gotok.Token) (int, []byte) {
	counters, err := db.GetUpdatesCount(t.Id)
	if err != nil && err != mgo.ErrNotFound {
		return Render(BackendError(err))
	}
	if err == mgo.ErrNotFound || len(counters) == 0 {
		return Render([]string{})
	}
	return Render(counters)
}

func GetUpdates(db DataBase, token *gotok.Token, pagination Pagination, context Context) (int, []byte) {
	t := context.Request.URL.Query().Get("type")
	if t == "" {
		return Render(ValidationError(errors.New("Blank type")))
	}
	updates, err := db.GetUpdates(token.Id, t, pagination)
	if err == mgo.ErrNotFound {
		return Render([]string{})
	}
	if err != nil {
		return Render(BackendError(err))
	}
	for _, u := range updates {
		if err := u.Prepare(context); err != nil {
			log.Println(err)
		}
	}
	return context.Render(updates)
}

func sendCounters(db DataBase, token *gotok.Token, realtime RealtimeInterface) error {
	counters, err := db.GetUpdatesCount(token.Id)
	if err != nil {
		return err
	}
	update := NewUpdate(token.Id, token.Id, "counters", Counters(counters))
	return realtime.Push(update)
}

func SetUpdatesRead(db DataBase, token *gotok.Token, req *http.Request, realtime RealtimeInterface) (int, []byte) {
	t := req.URL.Query().Get("type")
	if err := db.SetUpdatesRead(token.Id, t); err != nil {
		return Render(BackendError(err))
	}
	if err := sendCounters(db, token, realtime); err != nil {
		return Render(BackendError(err))
	}
	return Render(t)
}

func SetUpdateRead(db DataBase, token *gotok.Token, id bson.ObjectId, realtime RealtimeInterface) (int, []byte) {
	if err := db.SetUpdateRead(token.Id, id); err != nil {
		return Render(BackendError(err))
	}
	if err := sendCounters(db, token, realtime); err != nil {
		return Render(BackendError(err))
	}
	return Render("ok")
}

func Feedback(parser Parser, db DataBase, mail MailHtmlSender, token *gotok.Token) (int, []byte) {
	type Data struct {
		User  *User  `json:"-"`
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	v := new(Data)
	if err := parser.Parse(v); err != nil {
		return Render(ValidationError(err))
	}
	v.User = db.Get(token.Id)
	if err := mail.SendTo("feedback.html", feedbackEmail, fmt.Sprintf("Отзыв: %s", v.Title), v); err != nil {
		return Render(BackendError(err))
	}
	return Render(v)
}

func WantToTravel(parser Parser, db DataBase, mail MailHtmlSender, token *gotok.Token) (int, []byte) {
	type Data struct {
		User    *User  `json:"-"`
		Phone   string `json:"phone"`
		Country string `json:"country"`
	}
	v := new(Data)
	if err := parser.Parse(v); err != nil {
		return Render(ValidationError(err))
	}
	if !db.CountryExists(v.Country) {
		return Render(ValidationError(errors.New(fmt.Sprintf("Страна %s не существует", v.Country))))
	}
	v.User = db.Get(token.Id)
	if err := mail.SendTo("travel.html", feedbackEmail, fmt.Sprintf("%s - %s", v.User.Name, v.Country), v); err != nil {
		return Render(BackendError(err))
	}
	return Render(v)
}

func AddToken(db DataBase, parm martini.Params, t *gotok.Token) (int, []byte) {
	system := parm["system"]
	token := parm["token"]

	if token == "" {
		return Render(ValidationError(errors.New("Пустой токен")))
	}
	if system != "ios" && system != "android" {
		return Render(ValidationError(errors.New("Система должна быть ios или android")))
	}

	var err error
	if system == "ios" {
		err = db.AddIosToken(t.Id, token)
	} else {
		err = db.AddAndroidToken(t.Id, token)
	}

	if err != nil {
		return Render(BackendError(err))
	}

	return Render(token)
}

func RemoveToken(db DataBase, parm martini.Params, t *gotok.Token) (int, []byte) {
	system := parm["system"]
	token := parm["token"]

	if token == "" {
		return Render(ValidationError(errors.New("Пустой токен")))
	}
	if system != "ios" && system != "android" {
		return Render(ValidationError(errors.New("Система должна быть ios или android")))
	}

	var err error
	if system == "ios" {
		err = db.RemoveIosToken(t.Id, token)
	} else {
		err = db.RemoveAndroidToken(t.Id, token)
	}

	if err != nil {
		return Render(BackendError(err))
	}

	return Render(token)
}

func AddPresent(db DataBase, adapter StorageAdapter, req *http.Request, parser Parser, context Context) (int, []byte) {
	f, h, err := req.FormFile(FORM_FILE)
	type Data struct {
		Cost  int    `json:"cost"`
		Title string `json:"string`
	}
	data := new(Data)
	if err := parser.Parse(data); err != nil {
		return Render(ValidationError(err))
	}
	if data.Title == "" {
		return Render(ValidationError(errors.New("Bad title")))
	}
	if data.Cost < 0 {
		return Render(ValidationError(errors.New("Bad cost")))
	}
	format := "jpg"
	s := strings.Split(h.Header.Get(ContentTypeHeader), "/")
	if len(s) == 2 {
		format = s[1]
	}
	if err != nil {
		return Render(ValidationError(errors.New(fmt.Sprintf("No file at %s field", FORM_FILE))))
	}
	fid, _, _, err := adapter.Upload(f, "image", format)
	present := new(Present)
	present.Image = fid
	present.Title = data.Title
	present.Cost = data.Cost
	present.Time = time.Now()
	if err := db.AddPresent(present); err != nil {
		return Render(BackendError(err))
	}
	return context.Render(present)
}

func GetAllPresents(db DataBase, context Context) (int, []byte) {
	presents, err := db.GetAllPresents()
	if err != nil {
		return Render(BackendError(err))
	}
	return context.Render(Presents(presents))
}

func RemovePresent(db DataBase, id bson.ObjectId) (int, []byte) {
	if err := db.RemovePresent(id); err != nil {
		return Render(BackendError(err))
	}
	return Render("ok")
}

func AdminPresents(w http.ResponseWriter) (int, []byte) {
	data, err := AllTemplates.Bytes("presents.html")
	if err != nil {
		return Render(BackendError(err))
	}
	return http.StatusOK, data
}

func UpdatePresent(db DataBase, id bson.ObjectId, parser Parser, req *http.Request, adapter StorageAdapter, context Context) (int, []byte) {
	present := new(Present)
	f, h, ferr := req.FormFile(FORM_FILE)
	log.Println(req.Form, req.PostForm)
	if err := parser.Parse(present); err != nil {
		return Render(ValidationError(err))
	}
	if present.Title == "" {
		return Render(ValidationError(errors.New("Bad title")))
	}
	if present.Cost < 0 {
		return Render(ValidationError(errors.New("Bad cost")))
	}
	if ferr == nil {
		log.Println("file is updated")
		format := "jpg"
		s := strings.Split(h.Header.Get(ContentTypeHeader), "/")
		if len(s) == 2 {
			format = s[1]
		}
		fid, _, _, err := adapter.Upload(f, "image", format)
		if err != nil {
			return Render(BackendError(err))
		}
		present.Image = fid
	}
	p, err := db.UpdatePresent(id, present)
	if err != nil {
		return Render(err)
	}
	return context.Render(p)
}

func SendPresent(db DataBase, id bson.ObjectId, parm martini.Params, adapter StorageAdapter, parser Parser, token *gotok.Token) (int, []byte) {
	t := parm["title"]
	type Data struct {
		Text string `json:"text"`
	}
	data := new(Data)
	if err := parser.Parse(data); err != nil {
		return Render(ValidationError(err))
	}

	present, err := db.SendPresent(token.Id, id, t)
	if err != nil {
		return Render(BackendError(err))
	}

	return Render(present)
}

func GetUserPresents(db DataBase, id bson.ObjectId, adapter StorageAdapter, token *gotok.Token, context Context) (int, []byte) {
	presents, err := db.GetUserPresents(id)
	if err != nil && err != mgo.ErrNotFound {
		return Render(BackendError(err))
	}
	if len(presents) == 0 {
		return Render([]interface{}{})
	}

	return context.Render(PresentEvents(presents))
}

// Convert bytes to human readable string. Like a 2 MB, 64.2 KB, 52 B
func FormatBytes(i uint64) (result string) {
	switch {
	case i > (1024 * 1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f TB", float64(i)/1024/1024/1024/1024)
	case i > (1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f GB", float64(i)/1024/1024/1024)
	case i > (1024 * 1024):
		result = fmt.Sprintf("%#.02f MB", float64(i)/1024/1024)
	case i > 1024:
		result = fmt.Sprintf("%#.02f KB", float64(i)/1024)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	result = strings.Trim(result, " ")
	return
}

func GetSystemStatus(db DataBase) (int, []byte) {
	type System struct {
		Goroutines      int    `json:"goroutines"`
		Allocated       string `json:"allocated"`
		AllocatedHeap   string `json:"allocated_heap"`
		AllocatedTotal  string `json:"allocated_total"`
		Online          int    `json:"online"`
		RegisteredDay   int    `json:"registered_day"`
		RegisteredWeek  int    `json:"registered_week"`
		RegisteredMonth int    `json:"registered_month"`
		RegisteredYear  int    `json:"registered_year"`
		ActiveHour      int    `json:"active_hour"`
		ActiveDay       int    `json:"active_day"`
		ActiveWeek      int    `json:"active_week"`
	}

	data := new(System)
	data.Goroutines = runtime.NumGoroutine()
	mem := new(runtime.MemStats)
	runtime.ReadMemStats(mem)
	data.Allocated = FormatBytes(mem.Alloc)
	data.AllocatedHeap = FormatBytes(mem.HeapAlloc)
	data.AllocatedTotal = FormatBytes(mem.TotalAlloc)
	data.Online = db.Online()
	data.RegisteredDay = db.RegisteredCount(time.Hour * 24)
	data.RegisteredWeek = db.RegisteredCount(time.Hour * 24 * 7)
	data.RegisteredMonth = db.RegisteredCount(time.Hour * 24 * 30)
	data.RegisteredYear = db.RegisteredCount(time.Hour * 24 * 30 * 12)
	data.ActiveHour = db.ActiveCount(time.Hour)
	data.ActiveDay = db.ActiveCount(time.Hour * 24)
	data.ActiveWeek = db.ActiveCount(time.Hour * 24 * 7)
	return Render(data)
}

func Robots() (robots string) {
	robots = `User-agent: *
Disallow: /*openstat*
Disallow: /*utm*
Disallow: /*from*
Disallow: /*gclid*
Disallow: /register
Disallow: /login
Disallow: /search

Host: poputchiki.ru
Sitemap: http://poputchiki.ru/sitemap.xml
	`
	return robots
}

func Sitemap(db DataBase) string {
	items := []SitemapItem{}
	users := db.AllUsers()
	items = append(items, SitemapItem{"http://poputchiki.ru/"})
	items = append(items, SitemapItem{"http://poputchiki.ru/contacts/"})
	items = append(items, SitemapItem{"http://poputchiki.ru/terms/"})
	items = append(items, SitemapItem{"http://poputchiki.ru/about/"})
	for _, user := range users {
		items = append(items, SitemapItem{"http://poputchiki.ru/user/" + user.Id.Hex() + "/"})
	}
	sitemap, err := SitemapStr(items)
	if err != nil {
		return err.Error()
	}
	return sitemap
}

func AdvAdd(c Context) (int, []byte) {
	var (
		item    = new(StripeItem)
		request = new(StripeItemRequest)
		media   interface{}
	)
	if err := c.Parse(request); err != nil {
		return Render(ValidationError(err))
	}
	if len(request.Id.Hex()) != 0 {
		p, err := c.DB.GetPhoto(request.Id)
		if err == mgo.ErrNotFound {
			return Render(ErrObjectNotFound)
		}
		media = p
	}
	if err := c.DB.DecBalance(c.User.Id, adCost); err != nil {
		return Render(ErrorInsufficentFunds)
	}
	ad, err := c.DB.AddAdvertisement(c.User.Id, item, media)
	if err != nil {
		c.DB.IncBalance(c.User.Id, adCost)
		return Render(BackendError(err))
	}
	return c.Render(ad)
}

func AdvRemove(c Context, id bson.ObjectId) (int, []byte) {
	err := c.DB.RemoveAdvertisment(c.User.Id, id)
	if err == mgo.ErrNotFound {
		return Render(ErrObjectNotFound)
	}
	if err != nil {
		return Render(BackendError(err))
	}

	return Render("Removed")
}

func AdvGet(c Context, pagination Pagination) (int, []byte) {
	ads, err := c.DB.GetAds(pagination.Count, pagination.Offset)
	if err != nil {
		return Render(BackendError(err))
	}
	return c.Render(ads)
}

// init for random
func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}
