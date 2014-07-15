package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ernado/cymedia/photo"
	"github.com/ernado/gofbauth"
	"github.com/ernado/gorobokassa"
	"github.com/ernado/gosmsru"
	"github.com/ernado/gotok"
	"github.com/ernado/govkauth"
	. "github.com/ernado/poputchiki/models"
	"github.com/ernado/weed"
	"github.com/go-martini/martini"
	"github.com/rainycape/magick"
	"github.com/riobard/go-mailgun"
	"io"
	"labix.org/v2/mgo/bson"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	THUMB_SIZE               = 200
	PHOTO_MAX_SIZE           = 1000
	PHOTO_MAX_MEGABYTES      = 20
	VIDEO_MAX_MEGABYTES      = 50
	VIDEO_MAX_LENGTH_SECONDS = 360
	VIDEO_BITRATE            = 256
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
func GetUser(db DataBase, t *gotok.Token, id bson.ObjectId, webp WebpAccept, adapter *weed.Adapter) (int, []byte) {
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
	if t == nil || t.Id != id {
		user.CleanPrivate()
	}
	// preparing for rendering to json
	user.Prepare(adapter, db, webp)
	return Render(user)
}

// AddToFavorites adds target user to favorites of user
func AddToFavorites(db DataBase, id bson.ObjectId, r *http.Request) (int, []byte) {
	user := db.Get(id)
	// check user existance
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// get target user id from form
	hexId := r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}
	// convert it to bson.ObjectId
	favId := bson.ObjectIdHex(hexId)
	// get target user from database
	friend := db.Get(favId)
	// check for esistance
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	// add to favorites
	if err := db.AddToFavorites(user.Id, friend.Id); err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("updated")
}

// AddToBlacklist adds target user to blacklist of user
func AddToBlacklist(db DataBase, id bson.ObjectId, r *http.Request) (int, []byte) {
	user := db.Get(id)
	// check existance
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// get target id from form data
	hexId := r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}
	favId := bson.ObjectIdHex(hexId)
	friend := db.Get(favId)
	// check that target exists
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	if err := db.AddToBlacklist(user.Id, friend.Id); err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("added to blacklist")
}

// RemoveFromBlacklist removes target user from blacklist of another user
func RemoveFromBlacklist(db DataBase, id bson.ObjectId, r *http.Request) (int, []byte) {
	user := db.Get(id)
	// check existance
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	// get target id from form data
	hexId := r.FormValue(FORM_TARGET)
	if !bson.IsObjectIdHex(hexId) {
		return Render(ErrorBadId)
	}
	favId := bson.ObjectIdHex(hexId)
	friend := db.Get(favId)
	// check that target exists
	if friend == nil {
		return Render(ErrorUserNotFound)
	}
	if err := db.RemoveFromBlacklist(user.Id, friend.Id); err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("removed")
}

// RemoveFromFavorites removes target user from favorites of another user
func RemoveFromFavorites(db DataBase, id bson.ObjectId, r *http.Request) (int, []byte) {
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
	if err := db.RemoveFromFavorites(user.Id, friend.Id); err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("removed")
}

// GetFavorites returns list of users in favorites of target user
func GetFavorites(db DataBase, id bson.ObjectId, r *http.Request, webp WebpAccept, adapter *weed.Adapter) (int, []byte) {
	favorites := db.GetFavorites(id)
	// check for existance
	if favorites == nil {
		return Render([]interface{}{})
	}
	// clean private fields and prepare data
	for key, _ := range favorites {
		favorites[key].CleanPrivate()
		favorites[key].Prepare(adapter, db, webp)
	}
	return Render(favorites)
}

// GetFavorites returns list of users in guests of target user
func GetGuests(db DataBase, id bson.ObjectId, r *http.Request, webp WebpAccept, adapter *weed.Adapter) (int, []byte) {
	guests, err := db.GetAllGuests(id)
	if err != nil {
		return Render(ErrorBackend)
	}
	// check for existance
	if guests == nil {
		return Render([]interface{}{})
	}
	// clean private fields and prepare data
	for key, _ := range guests {
		guests[key].CleanPrivate()
		guests[key].Prepare(adapter, db, webp)
	}
	return Render(guests)
}

func AddToGuests(db DataBase, id bson.ObjectId, r *http.Request, realtime RealtimeInterface) (int, []byte) {
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

// Login checks the provided credentials and return token for user, setting appropriate
// auth cookies

type LoginCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(db DataBase, r *http.Request, w http.ResponseWriter, tokens gotok.Storage, parser Parser) (int, []byte) {
	credentials := &LoginCredentials{}
	if parser.Parse(credentials) != nil {
		return Render(ErrorBadRequest)
	}
	log.Println(credentials)
	username, password := credentials.Email, credentials.Password
	user := db.GetUsername(username)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	log.Println(user.Password, password, getHash(password, db.Salt()))
	if user.Password != getHash(password, db.Salt()) {
		return Render(ErrorAuth)
	}
	t, err := tokens.Generate(user.Id)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	http.SetCookie(w, t.GetCookie())
	return Render(t)
}

// Logout ends the current session and makes current token unusable
func Logout(db DataBase, r *http.Request, tokens gotok.Storage, t *gotok.Token) (int, []byte) {
	if err := tokens.Remove(t); err != nil {
		return Render(ErrorBackend)
	}
	return Render("logged out")
}

// Register checks the provided credentials, add new user with that credentials to
// database and returns new authorisation token, setting the appropriate cookies
func Register(db DataBase, r *http.Request, w http.ResponseWriter, tokens gotok.Storage) (int, []byte) {
	// load user data from form
	u := UserFromForm(r, db.Salt())
	// check that email is unique
	uDb := db.GetUsername(u.Email)
	if uDb != nil {
		log.Println(u.Email, "already registered")
		return Render(ErrorUserAlreadyRegistered) // todo: change error name
	}
	log.Println("registered", u.Password)
	// add to database
	if err := db.Add(u); err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	// generate token
	t, err := tokens.Generate(u.Id)
	if err != nil {
		return Render(ErrorBackend)
	}
	// generate confirmation token for email confirmation
	confTok := db.NewConfirmationToken(u.Id)
	if confTok == nil {
		return Render(ErrorBackend)
	}
	// anyncronously send email to user
	go func() {
		mgClient := mailgun.New(mailKey)
		message := ConfirmationMail{}
		message.Destination = u.Email
		message.Mail = "http://poputchiki.ru/api/confirm/email/" + confTok.Token
		log.Println("[email]", message.From(), message.To(), message.Text())
		_, err = mgClient.Send(message)
		log.Println(message)
		if err != nil {
			log.Println("[email]", err)
		}
	}()
	http.SetCookie(w, t.GetCookie())
	return Render(t)
}

// Update updates user information with provided key-value document
func Update(db DataBase, r *http.Request, id bson.ObjectId, parser Parser) (int, []byte) {
	query := bson.M{}
	// decoding json to map
	e := parser.Parse(&query)
	if e != nil {
		log.Println(e)
		return Render(ErrorBadRequest)
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
			log.Println(id, "was trying to edit", k)
		}
	}
	// checking field count
	if len(query) == 0 {
		log.Println("no fields to change")
		return Render(ErrorBadRequest)
	}

	// encoding to user - checking type & field existance
	user := &User{}
	err := convert(query, user)
	if err != nil {
		log.Println("unable to convert", err)
		return Render(ErrorBadRequest)
	}

	// encoding back to query object
	// marshalling to bson
	newQuery := bson.M{}
	tmp, err := bson.Marshal(user)
	if err != nil {
		log.Println("unable to marchal", err)
		return Render(ErrorBadRequest)
	}
	// unmarshalling to bson map
	err = bson.Unmarshal(tmp, &newQuery)
	if err != nil {
		log.Println("unable to unmarchal", err)
		return Render(ErrorBadRequest)
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
		log.Println(err)
		return Render(ErrorBackend)
	}
	// returning updated user
	return Render(db.Get(id))
}

func Must(err error) {
	if err != nil {
		log.Println(err)
	}
}

type MessageText struct {
	Text string `json:"text"`
}

func SendMessage(db DataBase, parser Parser, destination bson.ObjectId, r *http.Request, t *gotok.Token, realtime RealtimeInterface) (int, []byte) {
	message := &MessageText{}
	err := parser.Parse(message)
	if err != nil {
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	log.Println(err, message)
	text := message.Text
	origin := t.Id
	now := time.Now()

	if text == "" {
		return Render(ErrorBadRequest)
	}

	m1 := Message{bson.NewObjectId(), destination, origin, origin, destination, false, now, text}
	m2 := Message{bson.NewObjectId(), origin, destination, origin, destination, false, now, text}

	go func() {
		u := db.Get(destination)
		if u == nil {
			return
		}
		// check blacklist of destination
		for _, id := range u.Blacklist {
			if id == origin {
				Must(realtime.Push(origin, MessageSendBlacklisted{m1.Id}))
				return
			}
		}
		Must(realtime.Push(origin, m1))
		Must(realtime.Push(destination, m2))
		Must(db.AddMessage(&m1))
		Must(db.AddMessage(&m2))
	}()

	return Render("message sent")
}

func RemoveMessage(db DataBase, id bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
	message, err := db.GetMessage(id)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	if message.User != t.Id {
		return Render(ErrorNotAllowed)
	}
	go func() {
		Must(db.RemoveMessage(id))
	}()
	return Render("message removed")
}

func GetMessagesFromUser(db DataBase, origin bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
	messages, err := db.GetMessagesFromUser(t.Id, origin)
	if err != nil {
		return Render(ErrorBackend)
	}
	if messages == nil {
		return Render(ErrorUserNotFound)
	}
	return Render(messages)
}

func GetChats(db DataBase, id bson.ObjectId, webp WebpAccept, adapter *weed.Adapter) (int, []byte) {
	users, err := db.GetChats(id)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	for k := range users {
		users[k].Prepare(adapter, db, webp)
	}
	return Render(users)
}

func UploadVideo(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db DataBase, webpAccept WebpAccept, videoAccept VideoAccept, adapter *weed.Adapter) (int, []byte) {
	// c := weedo.NewClient(weedHost, weedPort)
	id := bson.NewObjectId()
	video := Video{Id: id, User: t.Id, Time: time.Now()}
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read from file", err)
		return Render(ErrorBackend)
	}

	length := r.ContentLength
	var b bytes.Buffer

	if length > 1024*1024*VIDEO_MAX_MEGABYTES {
		return Render(ErrorBadRequest)
	}

	sourceFilename := "/tmp/" + id.Hex() + "-source"
	buffer := make([]byte, 1024*300)
	f.Read(buffer)
	cmd := exec.Command("/bin/bash", "-c", "ffprobe "+sourceFilename)
	cmd.Stderr = bufio.NewWriter(&b)
	sourceFile, err := os.Create(sourceFilename)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	sourceFile.Write(buffer)
	sourceFile.Close()
	cmd.Start()
	cmd.Wait()

	// parsing duration
	testOutput := string(b.Bytes())
	durationToken := "Duration:"
	startDuration := strings.Index(testOutput, durationToken)
	endDuration := strings.Index(testOutput, ", start:")
	duration := strings.TrimSpace(testOutput[startDuration+len(durationToken) : endDuration])
	var hours, minutes, seconds, ms int64
	_, err = fmt.Sscanf(duration, "%02d:%02d:%02d.%02d", &hours, &minutes, &seconds, &ms)
	totalSeconds := hours*60*60 + minutes*60 + seconds
	video.Duration = totalSeconds

	// checking duration
	if duration == "N/A" || totalSeconds > VIDEO_MAX_LENGTH_SECONDS {
		os.Remove(sourceFilename)
		return Render(ErrorBadRequest)
	}

	sourceFile, err = os.OpenFile(sourceFilename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	// uploading full file
	_, err = io.Copy(sourceFile, f)
	sourceFile.Close()

	// pipes for processing and uploading
	uploadReaderMp4, uploadWriterMp4 := io.Pipe()
	uploadReaderWebm, uploadWriterWebm := io.Pipe()
	progressReaderMp4, progressWriterMp4 := io.Pipe()
	progressReaderWebm, progressWriterWebm := io.Pipe()

	// starting async processing
	wg := sync.WaitGroup{}
	wg.Add(3)
	// thumbnail goroutine
	go func() {
		log.Println("making screenshot")
		defer wg.Done()
		b := bytes.NewBuffer(nil)
		file, err := os.OpenFile(sourceFilename, os.O_RDONLY, 0644)
		if err != nil {
			log.Println(err)
			return
		}
		cmd := exec.Command("/bin/bash", "-c", "ffmpeg -i - -ss 00:00:01.00 -f image2 -vframes 1 -vcodec png -")
		cmd.Stdout = bufio.NewWriter(b)
		cmd.Stdin = file
		cmd.Start()
		cmd.Wait()
		file.Close()

		im, err := magick.Decode(bufio.NewReader(b))
		if err != nil {
			log.Println(err)
			return
		}
		thumbnail, err := im.CropToRatio(1.0, magick.CSCenter)
		if err != nil {
			return
		}
		thumbnail, err = thumbnail.Resize(THUMB_SIZE, THUMB_SIZE, magick.FBox)
		if err != nil {
			return
		}
		fid, purlWebp, _, err := uploadImageToWeed(adapter, thumbnail, "webp")
		if err != nil {
			log.Println("upload video", id, "error:", err)
			return
		}
		video.ThumbnailWebp = fid
		fid, purl, _, err := uploadImageToWeed(adapter, thumbnail, "jpeg")
		if err != nil {
			log.Println("upload video", id, "error:", err)
			return
		}
		video.ThumbnailJpeg = fid
		video.ThumbnailUrl = purl + ".jpeg"
		if webpAccept {
			video.ThumbnailUrl = purlWebp + ".webp"
		}
		log.Println("thumbnail generated", video.ThumbnailUrl)
	}()

	convert := func(format, command string, uploadWriter, progressWriter *io.PipeWriter) {
		log.Println(format, "convert started")
		defer wg.Done()
		defer uploadWriter.Close()
		defer progressWriter.Close()
		filename := fmt.Sprintf("%s.%s", id.Hex(), format)
		path := "/tmp/" + filename
		file, err := os.OpenFile(sourceFilename, os.O_RDONLY, 0644)
		if err != nil {
			log.Println(err)
			return
		}
		decodeReader := io.TeeReader(file, progressWriterWebm)
		defer progressWriter.Close()
		defer os.Remove(path)
		log.Println("executing", command)
		cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s %s", command, path))
		cmd.Stdin = decodeReader

		if e := cmd.Start(); e != nil {
			log.Println(e)
			return
		}
		cmd.Wait()
		log.Println(format, "ok")
		file.Close()
		file, err = os.Open(path)
		if err != nil {
			log.Println(err)
			return
		}

		io.Copy(uploadWriter, file)
		file.Close()
	}

	webpCmd := fmt.Sprintf("ffmpeg -i - -c:v libvpx -b:v %dk -c:a libvorbis -threads %d -vf crop=ih:ih,scale=%d:%d", VIDEO_BITRATE, processes, VIDEO_SIZE, VIDEO_SIZE)
	mp4Cmd := fmt.Sprintf("ffmpeg -i - -c:v h264 -c:a aac -b:v %dk -strict -2 -vf crop=ih:ih,scale=%d:%d", VIDEO_BITRATE, VIDEO_SIZE, VIDEO_SIZE)
	go convert("mp4", mp4Cmd, uploadWriterMp4, progressWriterMp4)
	go convert("webm", webpCmd, uploadWriterWebm, progressWriterWebm)

	// remove file after transcode
	go func() {
		c := make(chan bool, 1)
		go func() {
			wg.Wait()
			c <- true
		}()
		select {
		case <-c:
			{
				log.Println("transcoding ok")
			}
		case <-time.After(time.Second * 600):
			{
				log.Println("transcoding timed out")
			}
		}

		err := os.Remove(sourceFilename)
		if err != nil {
			log.Println("removing", sourceFilename, "failed:", err)
		}
	}()

	// progress report goroutine
	go func() {
		log.Println("processing progress")
		end := make(chan bool, 2)
		update := make(chan bool, 1)
		var pWebm float32
		var pMp4 float32
		start := func(writer *io.PipeWriter, reader *io.PipeReader, progress *float32) {
			log.Println("started progress processing")
			defer writer.Close()
			var read int64
			bufLen := length / 50
			for {
				cBytes, err := reader.Read(make([]byte, bufLen))
				if err == io.EOF {
					end <- true
					break
				}
				read += int64(cBytes)
				*progress = float32(read) / float32(length) * 100
				update <- true
			}
		}
		go func() {
			<-end
			<-end
			close(update)
			log.Println("updates channel closed")
		}()
		go start(progressWriterMp4, progressReaderMp4, &pMp4)
		go start(progressWriterWebm, progressReaderWebm, &pWebm)

		lastProgress := pWebm
		lastTime := time.Now()
		for _ = range update {
			progress := (pWebm + pMp4) / 2
			if (progress-lastProgress) > 1.0 && time.Since(lastTime) > time.Millisecond*300 {
				realtime.Push(t.Id, ProgressMessage{id, progress})
				lastTime = time.Now()
				lastProgress = progress
			}
		}
		realtime.Push(t.Id, ProgressMessage{id, 100.0})
		log.Println("progress hahdling finished")
	}()

	// upload goroutine
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(2)
		fileWebm := File{}
		fileMp4 := File{}
		var ok error
		ok = nil
		go func() {
			defer wg.Done()
			fid, purl, size, err := uploadToWeed(adapter, uploadReaderWebm, "video", "webm")
			if err != nil {
				ok = err
			}
			fileWebm = File{bson.NewObjectId(), fid, t.Id, time.Now(), "video/webm", size, purl}
		}()
		go func() {
			defer wg.Done()
			fid, purl, size, err := uploadToWeed(adapter, uploadReaderMp4, "video", "mp4")
			if err != nil {
				ok = err
			}
			fileMp4 = File{bson.NewObjectId(), fid, t.Id, time.Now(), "video/mp4", size, purl}
		}()
		wg.Wait()
		if ok != nil {
			log.Println(ok)
		} else {
			log.Println("uploaded")
		}
		video.VideoMpeg = fileMp4.Fid
		video.VideoWebm = fileWebm.Fid
		log.Println("video accept", videoAccept)
		if videoAccept == VaMp4 {
			video.VideoUrl = fileMp4.Url + ".mp4"
		}
		if videoAccept == VaWebm {
			video.VideoUrl = fileWebm.Url + ".webm"
		}
		log.Println("video transcoded", video)
		realtime.Push(t.Id, video)
	}()
	_, err = db.AddVideo(&video)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(video)
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

func pushProgress(length int64, rate int64, progressWriter *io.PipeWriter, progressReader *io.PipeReader, realtime RealtimeInterface, t *gotok.Token) {
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

func realtimeProgress(progress chan float32, realtime RealtimeInterface, t *gotok.Token) {
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

func uploadPhoto(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db DataBase, webpAccept WebpAccept, adapter *weed.Adapter) (*Photo, error) {
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
	go realtimeProgress(progress, realtime, t)
	p, err := uploader.Upload(length, f, progress)

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

func UploadPhoto(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db DataBase, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept, adapter)
	if err == ErrBadRequest {
		return Render(ErrorBadRequest)
	}
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(photo)
}

func AddStatus(db DataBase, id bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
	status := &StatusUpdate{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(status); err != nil {
		return Render(ErrorBadRequest)
	}
	status, err := db.AddStatus(t.Id, status.Text)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	err = db.DecBalance(t.Id, PromoCost)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(status)
}

func GetStatus(db DataBase, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetStatus(id)
	if err != nil {
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
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func UpdateStatus(db DataBase, id bson.ObjectId, r *http.Request, t *gotok.Token, parser Parser) (int, []byte) {
	status := &StatusUpdate{}
	if err := parser.Parse(status); err != nil {
		return Render(ErrorBadRequest)
	}
	status, err := db.UpdateStatusSecure(t.Id, id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func SearchPeople(db DataBase, pagination Pagination, r *http.Request, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	query, err := NewQuery(r.URL.Query())
	if err != nil {
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	result, err := db.Search(query, pagination.Count, pagination.Offset)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	for key, _ := range result {
		result[key].Prepare(adapter, db, webpAccept)
	}

	return Render(result)
}

func SearchStatuses(db DataBase, pagination Pagination, r *http.Request, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	query, err := NewQuery(r.URL.Query())
	if err != nil {
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	result, err := db.SearchStatuses(query, pagination.Count, pagination.Offset)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	for key, _ := range result {
		result[key].Prepare(adapter, webpAccept)
	}

	return Render(result)
}

func SearchPhoto(db DataBase, pagination Pagination, r *http.Request, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	query, err := NewQuery(r.URL.Query())
	if err != nil {
		log.Println(err)
		return Render(ErrorBadRequest)
	}
	result, err := db.SearchPhoto(query, pagination.Count, pagination.Offset)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	var video VideoAccept
	var audio AudioAccept

	for key, _ := range result {
		result[key].Prepare(adapter, webpAccept, video, audio)
	}

	return Render(result)
}

func AddStripeItem(db DataBase, t *gotok.Token, parser Parser) (int, []byte) {
	var media interface{}
	request := &StripeItemRequest{}
	if parser.Parse(request) != nil {
		return Render(ErrorBadRequest)
	}

	err := db.DecBalance(t.Id, PromoCost)
	if err != nil {
		return Render(ErrorInsufficentFunds)
	}
	switch request.Type {
	case "video":
		media = db.GetVideo(request.Id)
	case "audio":
		media = db.GetAudio(request.Id)
	case "photo":
		media, _ = db.GetPhoto(request.Id)
	default:
		return Render(ErrorBadRequest)
	}
	if media == nil {
		return Render(ErrorUserNotFound)
	}
	s, err := db.AddStripeItem(t.Id, media)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(s)
}

func GetStripe(db DataBase, adapter *weed.Adapter, pagination Pagination, webp WebpAccept, audio AudioAccept, video VideoAccept) (int, []byte) {
	stripe, err := db.GetStripe(pagination.Count, pagination.Offset)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	for _, v := range stripe {
		if err := v.Prepare(adapter, webp, video, audio); err != nil {
			return Render(ErrorBackend)
		}
	}
	return Render(stripe)
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
		return Render(ErrorBadRequest)
	}
	userToken, err := tokens.Generate(tok.User)
	if err != nil {
		return Render(ErrorBackend)
	}
	err = db.ConfirmPhone(userToken.Id)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
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
	db.NewConfirmationTokenValue(t.Id, token)
	u := db.Get(t.Id)
	if u == nil {
		return Render(ErrorBackend)
	}
	client := gosmsru.Client{}
	client.Key = smsKey
	phone := u.Phone
	err := client.Send(phone, codeStr)
	if err != nil {
		return Render(ErrorBadRequest)
	}
	return Render("ok")
}

func GetTransactionUrl(db DataBase, args martini.Params, t *gotok.Token, handler *TransactionHandler) (int, []byte) {
	value, err := strconv.Atoi(args["value"])
	if err != nil || value <= 0 {
		return Render(ErrorBadRequest)
	}

	url, transaction, err := handler.Start(t.Id, value, robokassaDescription)
	log.Println(value, transaction, gorobokassa.CRC(value, transaction.Id, robokassaPassword1))
	if err != nil {
		return Render(ErrorBackend)
	}

	return Render(url)
}

func RobokassaSuccessHandler(db DataBase, r *http.Request, handler *TransactionHandler) (int, []byte) {
	transaction, err := handler.Close(r)
	if err != nil {
		return Render(ErrorBadRequest)
	}

	err = db.IncBalance(transaction.User, uint(transaction.Value))
	if err != nil {
		log.Println("[CRITICAL ERROR]", "transaction", transaction)
		return Render(ErrorBadRequest)
	}
	return Render(transaction)
}

func LikeVideo(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.AddLikeVideo(t.Id, id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(db.GetVideo(id))
}

func RestoreLikeVideo(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveLikeVideo(t.Id, id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(db.GetVideo(id))
}

func GetLikersVideo(id bson.ObjectId, db DataBase, adapter *weed.Adapter, webp WebpAccept) (int, []byte) {
	likers := db.GetLikesVideo(id)
	for k := range likers {
		likers[k].Prepare(adapter, db, webp)
		likers[k].CleanPrivate()
	}
	return Render(likers)
}

func LikePhoto(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.AddLikePhoto(t.Id, id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(db.GetVideo(id))
}

func RestoreLikePhoto(t *gotok.Token, id bson.ObjectId, db DataBase) (int, []byte) {
	err := db.RemoveLikePhoto(t.Id, id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(db.GetVideo(id))
}

func GetLikersPhoto(id bson.ObjectId, db DataBase, adapter *weed.Adapter, webp WebpAccept) (int, []byte) {
	likers := db.GetLikesPhoto(id)
	for k := range likers {
		likers[k].Prepare(adapter, db, webp)
		likers[k].CleanPrivate()
	}
	return Render(likers)
}

func GetCountries(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	coutries, err := db.GetCountries(start)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(coutries)
}

func GetCities(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	country := req.URL.Query().Get("country")
	if country == "" {
		return Render(ErrorBadRequest)
	}
	cities, err := db.GetCities(start, country)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(cities)
}

func GetPlaces(db DataBase, req *http.Request) (int, []byte) {
	start := req.URL.Query().Get("start")
	places, err := db.GetPlaces(start)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(places)
}

func ForgotPassword(db DataBase, id bson.ObjectId) (int, []byte) {
	var err error
	u := db.Get(id)
	if u == nil {
		return Render(ErrorUserNotFound)
	}

	confTok := db.NewConfirmationToken(id)
	if confTok == nil {
		return Render(ErrorBackend)
	}
	// anyncronously send email to user
	go func() {
		mgClient := mailgun.New(mailKey)
		message := ConfirmationMail{}
		message.Destination = u.Email
		message.Mail = "http://poputchiki.ru/api/forgot/" + confTok.Token
		log.Println("[email]", message.From(), message.To(), message.Text())
		_, err = mgClient.Send(message)
		log.Println(message)
		if err != nil {
			log.Println("[email]", err)
		}
	}()

	return Render("ok")
}

func ResetPassword(db DataBase, r *http.Request, w http.ResponseWriter, args martini.Params, tokens gotok.Storage) {
	token := args["token"]
	if token == "" {
		code, data := Render(ErrorBadRequest)
		http.Error(w, string(data), code) // todo: set content-type
	}
	tok := db.GetConfirmationToken(token)
	if tok == nil {
		code, data := Render(ErrorBadRequest)
		http.Error(w, string(data), code) // todo: set content-type
	}
	userToken, err := tokens.Generate(tok.User)
	if err != nil {
		code, data := Render(ErrorBackend)
		http.Error(w, string(data), code) // todo: set content-type
	}
	err = db.ConfirmEmail(userToken.Id)
	if err != nil {
		log.Println(err)
		code, data := Render(ErrorBackend)
		http.Error(w, string(data), code) // todo: set content-type
	}
	http.SetCookie(w, userToken.GetCookie())
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func VkontakteAuthStart(r *http.Request, w http.ResponseWriter, client *govkauth.Client) {
	url := client.DialogURL()
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
}

func FacebookAuthStart(r *http.Request, w http.ResponseWriter, client *gofbauth.Client) {
	url := client.DialogURL()
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
}

func VkontakteAuthRedirect(db DataBase, r *http.Request, w http.ResponseWriter, tokens gotok.Storage, client *govkauth.Client) {
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
		newUser.Name, err = client.GetName(token.UserID)
		log.Println(newUser.Name, err)
		err = db.Add(newUser)
		if err != nil {
			code, _ := Render(ErrorBackend)
			http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
			return
		}
		u = db.GetUsername(token.Email)
	}
	userToken, err := tokens.Generate(u.Id)
	if err != nil {
		code, _ := Render(ErrorBackend)
		http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
		return
	}

	http.SetCookie(w, userToken.GetCookie())
	http.SetCookie(w, &http.Cookie{Name: "userId", Value: u.Id.Hex(), Path: "/"})
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func FacebookAuthRedirect(db DataBase, r *http.Request, w http.ResponseWriter, tokens gotok.Storage, client *gofbauth.Client) {
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
		log.Println(newUser.Name, err)
		err = db.Add(newUser)
		if err != nil {
			code, _ := Render(ErrorBackend)
			http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
			return
		}
		u = db.GetUsername(fbUser.Email)
	}
	userToken, err := tokens.Generate(u.Id)
	if err != nil {
		code, _ := Render(ErrorBackend)
		http.Error(w, "Серверная ошибка. Попробуйте позже", code) // todo: set content-type
		return
	}

	http.SetCookie(w, userToken.GetCookie())
	http.SetCookie(w, &http.Cookie{Name: "userId", Value: u.Id.Hex(), Path: "/"})
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// init for random
func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}
