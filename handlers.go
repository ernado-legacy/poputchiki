package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	conv "github.com/ernado/cymedia/mediad/models"
	"github.com/ernado/cymedia/mediad/query"
	"github.com/ernado/cymedia/photo"
	"github.com/ernado/gofbauth"
	// "github.com/ernado/gorobokassa"
	"github.com/ernado/gosmsru"
	"github.com/ernado/gotok"
	"github.com/ernado/govkauth"
	"github.com/ernado/poputchiki/activities"
	. "github.com/ernado/poputchiki/models"
	"github.com/ernado/weed"
	"github.com/go-martini/martini"
	"github.com/rainycape/magick"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
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
	ErrBadRequest = errors.New("bad request") // internal bad request error
)

// simple handler for testing the api from cyvisor
func Index() (int, []byte) {
	return Render("ok")
}

// GetUser handler for getting full user information
func GetUser(db DataBase, t *gotok.Token, id bson.ObjectId, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept, u Updater) (int, []byte) {
	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// checking for blacklist
	for _, u := range user.Blacklist {
		if u == t.Id {
			return Render(ErrorBlacklisted)
		}
	}
	// hiding private fields for non-owner
	if t.Id != id {
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
			u.Push(NewUpdate(id, t.Id, UpdateGuests, nil))
		}()
	}
	// preparing for rendering to json
	user.Prepare(adapter, db, webp, audio)
	user.SetIsFavorite(db.Get(t.Id))
	return Render(user)
}

func GetCurrentUser(db DataBase, t *gotok.Token, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept, u Updater) (int, []byte) {
	user := db.Get(t.Id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// checking for blacklist
	for _, u := range user.Blacklist {
		if u == t.Id {
			return Render(ErrorBlacklisted)
		}
	}
	// preparing for rendering to json
	user.Prepare(adapter, db, webp, audio)
	return Render(user)
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
		return Render(ErrorBadRequest)
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
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	return Render("removed")
}

// GetFavorites returns list of users in favorites of target user
func GetFavorites(db DataBase, id bson.ObjectId, r *http.Request, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept) (int, []byte) {
	favorites := db.GetFavorites(id)
	// check for existance
	if favorites == nil {
		return Render([]interface{}{})
	}
	// clean private fields and prepare data
	for key, _ := range favorites {
		favorites[key].CleanPrivate()
		favorites[key].Prepare(adapter, db, webp, audio)
	}
	return Render(favorites)
}

// GetFavorites returns list of users in favorites of target user
func GetFollowers(db DataBase, id bson.ObjectId, r *http.Request, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept, t *gotok.Token) (int, []byte) {
	favorites, err := db.GetAllUsersWithFavorite(id)
	if err != nil && err != mgo.ErrNotFound {
		return Render(BackendError(err))
	}
	// check for existance
	if favorites == nil {
		return Render([]interface{}{})
	}
	// clean private fields and prepare data
	Users(favorites).Prepare(adapter, db, webp, audio, db.Get(t.Id))
	return Render(favorites)
}

// GetFavorites returns list of users in favorites of target user
func GetBlacklisted(db DataBase, id bson.ObjectId, r *http.Request, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept, t *gotok.Token) (int, []byte) {
	blacklisted := db.GetBlacklisted(id)
	// check for existance
	if blacklisted == nil {
		return Render([]interface{}{})
	}
	Users(blacklisted).Prepare(adapter, db, webp, audio, db.Get(t.Id))
	return Render(blacklisted)
}

// GetFavorites returns list of users in guests of target user
func GetGuests(db DataBase, id bson.ObjectId, r *http.Request, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept, t *gotok.Token) (int, []byte) {
	guests, err := db.GetAllGuestUsers(id)
	if err != nil {
		return Render(BackendError(err))
	}
	// check for existance
	if guests == nil {
		return Render([]interface{}{})
	}
	// clean private fields and prepare data
	user := db.Get(t.Id)
	for key, _ := range guests {
		guests[key].CleanPrivate()
		guests[key].Prepare(adapter, db, webp, audio)
		guests[key].SetIsFavorite(user)
	}
	return Render(guests)
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
	username, password := credentials.Email, credentials.Password
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
func UpdateUser(db DataBase, r *http.Request, id bson.ObjectId, parser Parser, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept) (int, []byte) {
	query := bson.M{}
	// decoding json to map
	e := parser.Parse(&query)
	if e != nil {
		return Render(ValidationError(e))
	}

	// removes fields
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
		return Render(ValidationError(errors.New("no fields to change")))
	}

	// encoding to user - checking type & field existance
	user := &User{}
	err := convert(query, user)
	if err != nil {
		return Render(ValidationError(err))
	}

	if user.Password != "" {
		user.Password = getHash(user.Password, db.Salt())
	}

	// encoding back to query object
	// marshalling to bson
	newQuery := bson.M{}
	tmp, err := bson.Marshal(user)
	if err != nil {
		return Render(ValidationError(err))
	}
	// unmarshalling to bson map
	err = bson.Unmarshal(tmp, &newQuery)
	if err != nil {
		return Render(ValidationError(err))
	}

	// removing fields, that dont exist in initial query
	for k := range newQuery {
		_, ok := query[k]
		if !ok {
			delete(newQuery, k)
		}
	}
	// updating user
	_, err = db.Update(id, newQuery)
	if err != nil {
		return Render(BackendError(err))
	}
	// returning updated user
	updated := db.Get(id)
	updated.Prepare(adapter, db, webp, audio)
	return Render(updated)
}

func Must(err error) {
	if err != nil {
		log.Println(err)
	}
}

type MessageText struct {
	Text string `json:"text"`
}

func SendMessage(db DataBase, parser Parser, destination bson.ObjectId, r *http.Request, t *gotok.Token, realtime AutoUpdater) (int, []byte) {
	message := &MessageText{}
	err := parser.Parse(message)
	if err != nil {
		return Render(ValidationError(err))
	}

	text := message.Text
	origin := t.Id

	if text == "" {
		return Render(ValidationError(errors.New("Blank text")))
	}

	m1, m2 := NewMessagePair(origin, destination, text)
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
	return Render("message sent")
}

func SendInvite(db DataBase, parser Parser, engine activities.Handler, destination bson.ObjectId, r *http.Request, t *gotok.Token, realtime AutoUpdater) (int, []byte) {
	text := "Вас пригласили в путешествие"
	origin := t.Id
	now := time.Now()

	m1 := &Message{bson.NewObjectId(), destination, origin, origin, destination, false, now, text, true}
	m2 := &Message{bson.NewObjectId(), origin, destination, origin, destination, false, now, text, true}

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
	Must(realtime.Push(origin, m1))
	Must(realtime.Push(destination, m2))
	Must(db.AddMessage(m1))
	Must(db.AddMessage(m2))
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

func GetMessagesFromUser(db DataBase, origin bson.ObjectId, r *http.Request, t *gotok.Token, pagination Pagination) (int, []byte) {
	messages, err := db.GetMessagesFromUser(t.Id, origin, pagination)
	if err != nil {
		return Render(BackendError(err))
	}
	if messages == nil {
		return Render([]interface{}{})
	}
	err = db.SetReadMessagesFromUser(t.Id, origin)
	if err != nil {
		log.Println("SetReadMessagesFromUser", err)
	}
	return Render(messages)
}

func GetChats(db DataBase, id bson.ObjectId, webp WebpAccept, adapter *weed.Adapter, audio AudioAccept) (int, []byte) {
	dialogs, err := db.GetChats(id)
	if err != nil {
		return Render(BackendError(err))
	}
	for k := range dialogs {
		dialogs[k].User.Prepare(adapter, db, webp, audio)
		dialogs[k].User.CleanPrivate()

		dialogs[k].OriginUser.Prepare(adapter, db, webp, audio)
		dialogs[k].OriginUser.CleanPrivate()
	}
	return Render(dialogs)
}

// reads data from io.Reader, uploads it with type/format and returs fid, purl and error
func uploadToWeed(adapter *weed.Adapter, reader io.Reader, t, format string) (string, string, int64, error) {
	fid, size, err := adapter.Client().AssignUpload(t+"."+format, t+"/"+format, reader)
	if err != nil {
		log.Println(t, format, err)
		return "", "", size, err
	}
	purl, err := adapter.GetUrl(fid)
	if err != nil {
		log.Println(err)
		return "", "", size, err
	}
	log.Println(t, format, "uploaded", purl)
	return fid, purl, size, nil
}

func uploadImageToWeed(adapter *weed.Adapter, image *magick.Image, format string) (string, string, int64, error) {
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

func uploadPhoto(r *http.Request, t *gotok.Token, realtime AutoUpdater, db DataBase, webpAccept WebpAccept, adapter *weed.Adapter) (*Photo, error) {
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read form file", err)
		return nil, ErrBadRequest
	}

	length := r.ContentLength
	if length > 1024*1024*PHOTO_MAX_MEGABYTES {
		return nil, ErrBadRequest
	}

	uploader := photo.NewUploader(adapter, PHOTO_MAX_SIZE, THUMB_SIZE)
	progress := make(chan float32)
	go func() {
		for _ = range progress {
			continue
		}
	}()
	p, err := uploader.Upload(length, f, progress)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	c := func(input *photo.File) File {
		output := &File{}
		output.Id = bson.NewObjectId()
		output.User = t.Id
		convert(input, output)
		db.AddFile(output)
		return *output
	}

	newPhoto, err := db.AddPhoto(t.Id, c(&p.ImageJpeg), c(&p.ImageWebp), c(&p.ThumbnailJpeg), c(&p.ThumbnailWebp), "")
	newPhoto.ImageUrl = p.ImageJpeg.Url
	newPhoto.ThumbnailUrl = p.ThumbnailJpeg.Url

	if bool(webpAccept) {
		newPhoto.ImageUrl = p.ImageWebp.Url
		newPhoto.ThumbnailUrl = p.ThumbnailWebp.Url
	}
	return newPhoto, err
}

func UploadPhoto(r *http.Request, engine activities.Handler, t *gotok.Token, realtime AutoUpdater, db DataBase, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept, adapter)
	if err == ErrBadRequest {
		return Render(ValidationError(err))
	}
	if err != nil {
		return Render(BackendError(err))
	}
	engine.Handle(activities.Photo)
	return Render(photo)
}

func AddStatus(db DataBase, r *http.Request, t *gotok.Token, parser Parser, engine activities.Handler) (int, []byte) {
	status := new(Status)
	if err := parser.Parse(status); err != nil {
		return Render(ValidationError(err))
	}

	count, err := db.GetLastDayStatusesAmount(t.Id)
	if err != nil {
		return Render(BackendError(err))
	}

	allowed := statusesPerDay
	u := db.Get(t.Id)
	if u.Vip {
		allowed = statusesPerDayVip
	}
	if count >= allowed {
		return Render(ErrorInsufficentFunds)
	}
	status, err = db.AddStatus(t.Id, status.Text)
	if err != nil {
		go db.IncBalance(t.Id, PromoCost)
		return Render(BackendError(err))
	}
	engine.Handle(activities.Status)
	return Render(status)
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

func SearchPeople(db DataBase, pagination Pagination, r *http.Request, t *gotok.Token, webpAccept WebpAccept, adapter *weed.Adapter, audio AudioAccept) (int, []byte) {
	q := addGeo(db, t, r)
	query, err := NewQuery(q)
	if err != nil {
		return Render(ValidationError(err))
	}
	if err := query.Validate(db); err != nil {
		return Render(ValidationError(err))
	}
	u := db.Get(t.Id)
	if !u.Vip {
		query.Sponsor = ""
	}
	result, count, err := db.Search(query, pagination)
	if err != nil {
		return Render(BackendError(err))
	}
	Users(result).Prepare(adapter, db, webpAccept, audio, u)
	return Render(SearchResult{result, count})
}

func SearchStatuses(db DataBase, pagination Pagination, r *http.Request, t *gotok.Token, webpAccept WebpAccept, audio AudioAccept, adapter *weed.Adapter) (int, []byte) {
	q := addGeo(db, t, r)
	query, err := NewQuery(q)
	if err != nil {
		return Render(ValidationError(err))
	}
	if err := query.Validate(db); err != nil {
		return Render(ValidationError(err))
	}
	u := db.Get(t.Id)
	if !u.Vip {
		query.Sponsor = ""
	}
	result, err := db.SearchStatuses(query, pagination)
	if err != nil {
		return Render(BackendError(err))
	}
	user := db.Get(t.Id)
	for key, _ := range result {
		result[key].Prepare(db, adapter, webpAccept, audio)
		result[key].UserObject.CleanPrivate()
		result[key].UserObject.SetIsFavorite(user)
	}

	return Render(result)
}

func SearchPhoto(db DataBase, pagination Pagination, r *http.Request, t *gotok.Token, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
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

	var video VideoAccept
	var audio AudioAccept

	for key, _ := range result {
		result[key].Prepare(adapter, webpAccept, video, audio)
		if result[key].UserObject != nil {
			result[key].UserObject.Prepare(adapter, db, webpAccept, audio)
			result[key].UserObject.CleanPrivate()
		}
	}

	return Render(result)
}

func GetUserPhoto(db DataBase, id bson.ObjectId, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	photo, err := db.GetUserPhoto(id)
	if err != nil {
		return Render(ErrorBackend)
	}
	var video VideoAccept
	var audio AudioAccept

	for key := range photo {
		photo[key].Prepare(adapter, webpAccept, video, audio)
	}

	return Render(photo)
}

func AddStripeItem(engine activities.Handler, db DataBase, t *gotok.Token, parser Parser, adapter *weed.Adapter, pagination Pagination, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	var media interface{}
	user := db.Get(t.Id)
	request := new(StripeItemRequest)
	if err := parser.Parse(request); err != nil {
		return Render(ValidationError(err))
	}
	i := new(StripeItem)
	i.Id = bson.NewObjectId()
	i.User = t.Id
	if !*development {
		err := db.DecBalance(t.Id, PromoCost)
		if err != nil {
			return Render(ErrorInsufficentFunds)
		}
	}
	i.Type = request.Type
	switch request.Type {
	case "video":
		video := db.GetVideo(request.Id)
		if video == nil {
			return Render(ErrorObjectNotFound)
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
			return Render(ErrorObjectNotFound)
		}
		i.ImageJpeg = user.AvatarJpeg
		i.ImageWebp = user.AvatarWebp
		media = audio
	case "photo":
		p, err := db.GetPhoto(request.Id)
		if err != nil && err != mgo.ErrNotFound {
			return Render(BackendError(err))
		}
		if p == nil {
			return Render(ErrorObjectNotFound)
		}
		i.ImageJpeg = p.ThumbnailJpeg
		i.ImageWebp = p.ThumbnailWebp
		media = p
	default:
		return Render(ValidationError(errors.New("Bad type")))
	}
	if media == nil {
		return Render(ErrorObjectNotFound)
	}
	s, err := db.AddStripeItem(i, media)
	if err != nil {
		return Render(BackendError(err))
	}
	engine.Handle(activities.Promo)
	s.Prepare(db, adapter, webp, video, audio)
	return Render(s)
}

func GetStripe(db DataBase, adapter *weed.Adapter, pagination Pagination, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	stripe, err := db.GetStripe(pagination.Count, pagination.Offset)
	if err != nil {
		return Render(BackendError(err))
	}
	for _, v := range stripe {
		if err := v.Prepare(db, adapter, webp, video, audio); err != nil {
			log.Println(err)
			// return Render(ErrorBackend)
		}
	}
	return Render(stripe)
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
func ConfirmEmail(db DataBase, args martini.Params, w http.ResponseWriter, tokens gotok.Storage) (int, []byte) {
	token := args["token"]
	if token == "" {
		return Render(ErrorBadRequest)
	}
	tok := db.GetConfirmationToken(token)
	if tok == nil {
		return Render(ErrorBadRequest)
	}
	userToken, err := tokens.Generate(tok.User)
	if err != nil {
		return Render(ErrorBackend)
	}
	err = db.ConfirmEmail(userToken.Id)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	http.SetCookie(w, userToken.GetCookie())
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
	value, err := strconv.Atoi(args["value"])
	if err != nil || value <= 0 {
		code, data := Render(ErrorBadRequest)
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
	transaction, err := handler.Close(r)
	if err != nil {
		return Render(ValidationError(err))
	}
	err = db.IncBalance(transaction.User, uint(transaction.Value))
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
	go u.Push(NewUpdate(v.User, t.Id, UpdateLikes, v))
	return Render(v)
}

func RestoreLikeVideo(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveLikeVideo(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	return Render(db.GetVideo(id))
}

func GetLikersVideo(id bson.ObjectId, db DataBase, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, t *gotok.Token) (int, []byte) {
	likers := db.GetLikesVideo(id)
	// check for existance
	if likers == nil {
		return Render([]interface{}{})
	}
	Users(likers).Prepare(adapter, db, webp, audio, db.Get(t.Id))
	return Render(likers)
}

func LikePhoto(t *gotok.Token, id bson.ObjectId, db DataBase, engine activities.Handler, u Updater) (int, []byte) {
	err := db.AddLikePhoto(t.Id, id)
	if err != nil {
		return Render(BackendError(err))
	}
	engine.Handle(activities.Like)
	p, _ := db.GetPhoto(id)
	go u.Push(NewUpdate(p.User, t.Id, UpdateLikes, p))
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

func RemovePhoto(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemovePhoto(t.Id, id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	if err != nil {
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

func GetLikersPhoto(id bson.ObjectId, db DataBase, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, t *gotok.Token) (int, []byte) {
	likers := db.GetLikesPhoto(id)
	// check for existance
	if likers == nil {
		return Render([]interface{}{})
	}
	Users(likers).Prepare(adapter, db, webp, audio, db.Get(t.Id))
	return Render(likers)
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
	log.Println(u.Push(NewUpdate(s.User, t.Id, UpdateLikes, s)))
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

func GetLikersStatus(id bson.ObjectId, db DataBase, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, t *gotok.Token) (int, []byte) {
	likers := db.GetLikesStatus(id)
	// check for existance
	if likers == nil {
		return Render([]interface{}{})
	}
	Users(likers).Prepare(adapter, db, webp, audio, db.Get(t.Id))
	return Render(likers)
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

func ForgotPassword(db DataBase, args martini.Params, mail MailHtmlSender) (int, []byte) {
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
			Url string
		}
		data := Data{"http://poputchiki.ru/api/forgot/" + confTok.Token}
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

func ExportPhoto(db DataBase, id bson.ObjectId, adapter *weed.Adapter, url string) *Photo {
	res, err := http.Get(url)
	f := res.Body
	if err != nil {
		log.Println("unable to read form file", err)
		return nil
	}
	length := res.ContentLength
	uploader := photo.NewUploader(adapter, PHOTO_MAX_SIZE, THUMB_SIZE)
	progress := make(chan float32)
	go func() {
		for _ = range progress {
			continue
		}
	}()

	p, err := uploader.Upload(length, f, progress)

	c := func(input *photo.File) File {
		output := &File{}
		output.Id = bson.NewObjectId()
		output.User = id
		convert(input, output)
		db.AddFile(output)
		return *output
	}

	defer func() {
		err := recover()
		if err != nil {
			log.Println(err)
		}
	}()

	newPhoto, err := db.AddPhoto(id, c(&p.ImageJpeg), c(&p.ImageWebp), c(&p.ThumbnailJpeg), c(&p.ThumbnailWebp), "")
	if err != nil {
		log.Println(err)
	}
	return newPhoto
}

func ExportThumbnail(adapter *weed.Adapter, fid string) (thumbnailJpeg, thumbnailWebp string, err error) {
	url, err := adapter.GetUrl(fid)
	if err != nil {
		return
	}
	res, err := http.Get(url)
	if err != nil {
		log.Println("unable to read form file", err)
		return
	}
	f := res.Body
	length := res.ContentLength
	uploader := photo.NewUploader(adapter, PHOTO_MAX_SIZE, THUMB_SIZE)
	progress := make(chan float32)
	go func() {
		for _ = range progress {
			continue
		}
	}()

	p, err := uploader.Upload(length, f, progress)

	if err != nil {
		return
	}

	defer func() {
		rerr := recover()
		if rerr != nil {
			err = errors.New("export failed")
			return
		}
	}()

	return p.ThumbnailJpeg.Fid, p.ThumbnailWebp.Fid, nil
}

func VkontakteAuthRedirect(db DataBase, r *http.Request, w http.ResponseWriter, adapter *weed.Adapter, tokens gotok.Storage, client *govkauth.Client) {
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
		log.Println(newUser.Name, err)
		err = db.Add(newUser)
		if err != nil {
			code, _ := Render(ErrorBackend)
			http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
			return
		}
		u = db.GetUsername(token.Email)

		if user.Photo != "" {
			p := ExportPhoto(db, u.Id, adapter, user.Photo)
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

func FacebookAuthRedirect(db DataBase, r *http.Request, adapter *weed.Adapter, w http.ResponseWriter, tokens gotok.Storage, client *gofbauth.Client) {
	token, err := client.GetAccessToken(r)
	if err != nil {
		code, _ := Render(ErrorBadRequest)
		http.Error(w, "Авторизация невозможна", code)
		return
	}
	fbUser, err := client.GetUser(token.AccessToken)
	log.Println(err)
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
		newUser.Birthday = fbUser.Birthday
		newUser.Rating = 100.0
		log.Println(newUser.Name, err)
		err = db.Add(newUser)
		if err != nil {
			code, _ := Render(ErrorBackend)
			http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
			return
		}
		u = db.GetUsername(fbUser.Email)
		log.Println(fbUser.Photo)
		if fbUser.Photo != "" {
			p := ExportPhoto(db, u.Id, adapter, fbUser.Photo)
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
	cookie, err := r.Cookie("admin")
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	token, err := tokens.Get(cookie.Value)
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	user := db.Get(token.Id)
	if user == nil || !user.IsAdmin {
		code, data := Render(ErrorAuth)
		http.Error(w, string(data), code)
		return
	}
	view, err := template.ParseFiles("static/html/index.html")
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	view.Execute(w, nil)
}

func AdminLogin(id bson.ObjectId, t *gotok.Token, db DataBase, w http.ResponseWriter, r *http.Request, tokens gotok.Storage) {
	user := db.Get(t.Id)
	if user == nil || !user.IsAdmin {
		code, data := Render(ErrorAuth)
		http.Error(w, string(data), code)
		return
	}

	userToken, err := tokens.Generate(id)
	if err != nil {
		code, data := Render(BackendError(err))
		http.Error(w, string(data), code)
		return
	}

	http.SetCookie(w, userToken.GetCookie())
	http.SetCookie(w, &http.Cookie{Name: "userId", Value: id.Hex(), Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "admin", Value: userToken.Token, Path: "/"})
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func UploadVideoFile(r *http.Request, client query.QueryClient, db DataBase, adapter *weed.Adapter, t *gotok.Token) (int, []byte) {
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
	optsMpeg.Duration = 10
	optsMpeg.Video.Height = VIDEO_SIZE
	optsMpeg.Video.Width = VIDEO_SIZE

	optsWebm := new(conv.VideoOptions)
	optsWebm.Video.Format = "libvpx"
	optsWebm.Audio.Format = "libvorbis"
	optsWebm.Audio.Bitrate = AUDIO_BITRATE
	optsWebm.Video.Bitrate = VIDEO_BITRATE
	optsWebm.Video.Square = true
	optsWebm.Duration = 10
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

func GetUserVideo(db DataBase, id bson.ObjectId, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	v, err := db.GetUserVideo(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	VideoSlice(v).Prepare(adapter, webp, video, audio)
	return Render(v)
}

func GetUserMedia(db DataBase, id bson.ObjectId, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	v, err := db.GetUserVideo(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	VideoSlice(v).Prepare(adapter, webp, video, audio)
	p, err := db.GetUserPhoto(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	PhotoSlice(p).Prepare(adapter, webp, video, audio)
	return Render(MakeMediaSlice(p, v))
}

func GetVideo(db DataBase, id bson.ObjectId, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	v := db.GetVideo(id)
	if v == nil {
		return Render(ErrorObjectNotFound)
	}
	if err := v.Prepare(adapter, webp, video, audio); err != nil {
		log.Println(err)
	}
	return Render(v)
}

func GetPhoto(db DataBase, id bson.ObjectId, adapter *weed.Adapter, webp WebpAccept) (int, []byte) {
	p, err := db.GetPhoto(id)
	if err == mgo.ErrNotFound {
		return Render(ErrorObjectNotFound)
	}
	var (
		video VideoAccept
		audio AudioAccept
	)
	if err := p.Prepare(adapter, webp, video, audio); err != nil {
		return Render(ErrorBackend)
	}
	return Render(p)
}

func UploadAudio(r *http.Request, client query.QueryClient, db DataBase, adapter *weed.Adapter, t *gotok.Token) (int, []byte) {
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
	optsAac := &conv.AudioOptions{Bitrate: AUDIO_BITRATE, Format: "aac"}
	optsAac.Duration = 10
	optsVorbis := &conv.AudioOptions{Bitrate: AUDIO_BITRATE, Format: "libvorbis"}
	optsVorbis.Duration = 10
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

func GetUpdates(db DataBase, token *gotok.Token, pagination Pagination, req *http.Request, adapter *weed.Adapter, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	t := req.URL.Query().Get("type")
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
		if err := u.Prepare(db, adapter, webp, video, audio); err != nil {
			log.Println(err)
		}
	}
	return Render(updates)
}

func SetUpdateRead(db DataBase, token *gotok.Token, id bson.ObjectId) (int, []byte) {
	if err := db.SetUpdateRead(token.Id, id); err != nil {
		return Render(BackendError(err))
	}
	return Render("ok")
}

// init for random
func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}
