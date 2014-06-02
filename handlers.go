package main

import (
	// "bytes"
	"encoding/json"
	"github.com/ginuerzh/weedo"
	"github.com/rainycape/magick"
	// "github.com/go-martini/martini"
	"io"
	"labix.org/v2/mgo/bson"
	"log"
	"os"
	"os/exec"
	// "strings"
	// "mime/multipart"
	"errors"
	"net/http"
	"sync"
	"time"
)

const (
	THUMB_SIZE               = 200
	PHOTO_MAX_SIZE           = 1000
	PHOTO_MAX_MEGABYTES      = 20
	VIDEO_MAX_MEGABYTES      = 50
	VIDEO_MAX_LENGTH_SECONDS = 60
	JSON_HEADER              = "application/json; charset=utf-8"
	WEBP                     = "webp"
	JPEG                     = "jpeg"
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
)

func ReadJson(r *http.Request, i *interface{}) {
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(i)
}

func GetUser(db UserDB, t *Token, id bson.ObjectId) (int, []byte) {
	log.Println("get user", id.Hex())
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

func AddToFavorites(db UserDB, id bson.ObjectId, r *http.Request) (int, []byte) {
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

func AddToBlacklist(db UserDB, id bson.ObjectId, r *http.Request) (int, []byte) {
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

func RemoveFromBlacklist(db UserDB, id bson.ObjectId, r *http.Request) (int, []byte) {
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

func RemoveFromFavorites(db UserDB, id bson.ObjectId, r *http.Request) (int, []byte) {
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

func GetFavorites(db UserDB, id bson.ObjectId, r *http.Request) (int, []byte) {
	favorites := db.GetFavorites(id)
	if favorites == nil {
		return Render(ErrorUserNotFound)
	}

	for key, _ := range favorites {
		favorites[key].CleanPrivate()
	}

	return Render(favorites)
}

func GetGuests(db UserDB, id bson.ObjectId, r *http.Request) (int, []byte) {
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

func AddToGuests(db UserDB, id bson.ObjectId, r *http.Request, realtime RealtimeInterface) (int, []byte) {
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
	log.Println(user)
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

func Logout(db UserDB, r *http.Request, tokens TokenStorage, t *Token) (int, []byte) {
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

func Update(db UserDB, r *http.Request, id bson.ObjectId) (int, []byte) {
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

func SendMessage(db UserDB, id bson.ObjectId, r *http.Request, t *Token, realtime RealtimeInterface) (int, []byte) {
	text := r.FormValue(FORM_TEXT)

	if text == BLANK {
		return Render(ErrorBadRequest)
	}

	destination := id
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

func RemoveMessage(db UserDB, id bson.ObjectId, r *http.Request, t *Token) (int, []byte) {
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

func GetMessagesFromUser(db UserDB, origin bson.ObjectId, r *http.Request, t *Token) (int, []byte) {
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

func UploadVideo(r *http.Request, t *Token, realtime RealtimeInterface) (int, []byte) {
	c := weedo.NewClient(weedHost, weedPort)
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read from file", err)
		return Render(ErrorBackend)
	}

	length := r.ContentLength
	if length > 1024*1024*VIDEO_MAX_MEGABYTES {
		return Render(ErrorBadRequest)
	}

	progressReader, progressWriter := io.Pipe()
	uploadReader, uploadWriter := io.Pipe()
	decodeReader := io.TeeReader(f, progressWriter)

	// decoding goroutine
	go func() {
		defer progressWriter.Close()
		defer uploadWriter.Close()
		cmd := exec.Command("/bin/bash", "-c", "ffmpeg -i - -c:v libvpx -b:v 256k -c:a libvorbis -cpu-used 4 -vf scale=300:ih*300/iw,crop=out_w=in_h -f webm - ")
		cmd.Stdin = decodeReader
		cmd.Stderr = os.Stdout
		cmd.Stdout = uploadWriter
		e := cmd.Start()
		if e != nil {
			log.Println(e)
		}
		cmd.Wait()
		log.Println("ok")
	}()

	// download progress goroutine
	go pushProgress(length, progressWriter, progressReader, realtime, t)

	fid, purl, err := uploadToWeed(c, uploadReader, "video", "webp")
	if err != nil {
		return Render(ErrorBackend)
	}
	im := Image{bson.NewObjectId(), fid, purl}
	return Render(im)
}

// reads data from io.Reader, uploads it with type/format and returs fid, purl and error
func uploadToWeed(c *weedo.Client, reader io.Reader, t, format string) (string, string, error) {
	fid, _, err := c.AssignUpload(t+"."+format, t+"/"+format, reader)
	if err != nil {
		log.Println(t, format, err)
		return "", "", err
	}
	purl, _, err := c.GetUrl(fid)
	if err != nil {
		log.Println(err)
		return "", "", err
	}
	log.Println(t, format, "uploaded", purl)
	return fid, purl, nil
}

func uploadImageToWeed(c *weedo.Client, image *magick.Image, format string) (string, string, error) {
	encodeReader, encodeWriter := io.Pipe()
	go func() {
		defer encodeWriter.Close()
		info := magick.NewInfo()
		info.SetFormat(format)
		err := image.Encode(encodeWriter, info)
		if err != nil {
			log.Println(err)
		}
	}()
	return uploadToWeed(c, encodeReader, "image", format)
}

func pushProgress(length int64, progressWriter *io.PipeWriter, progressReader *io.PipeReader, realtime RealtimeInterface, t *Token) {
	defer progressWriter.Close()
	var p float32
	var read int64
	bufLen := length / 50
	for {
		buffer := make([]byte, bufLen)
		cBytes, err := progressReader.Read(buffer)
		if err == io.EOF {
			break
		}
		read = read + int64(cBytes)
		//fmt.Printf("read: %v \n",read )
		p = float32(read) / float32(length) * 100
		if t != nil {
			realtime.Push(t.Id, ProgressMessage{p})
		}
	}
}

func uploadPhoto(r *http.Request, t *Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept) (*Photo, error) {
	c := weedo.NewClient(weedHost, weedPort)
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read form file", err)
		return nil, err
	}

	length := r.ContentLength
	if length > 1024*1024*PHOTO_MAX_SIZE {
		return nil, errors.New("bad request")
	}

	progressReader, progressWriter := io.Pipe()
	decodeReader := io.TeeReader(f, progressWriter)

	// download progress goroutine
	go pushProgress(length, progressWriter, progressReader, realtime, t)

	// trying to decode image while receiving it
	im, err := magick.Decode(decodeReader)
	if err != nil {
		return nil, err
	}

	height := float64(im.Height())
	width := float64(im.Width())
	max := float64(PHOTO_MAX_SIZE)

	// calculating scale-down ratio
	ratio := max / width
	if height > width {
		ratio = max / height
	}

	// image dimensions is smaller than maximum
	if height < max && width < max {
		ratio = 1.0
	}

	// preparing variables for concurrent uploading/processing
	failed := false
	var photoWebp, photoJpeg File
	var purlJpeg, purlWebp string
	var thumbWebp, thumbJpeg File
	var thumbPurlJpeg, thumbPurlWebp string

	wg := new(sync.WaitGroup)
	wg.Add(6)

	// generating abpstract upload function
	upload := func(image *magick.Image, url *string, photo *File, extension, format string) {
		defer wg.Done()
		fid, purl, err := uploadImageToWeed(c, image, extension)
		*url = purl
		if err != nil {
			failed = true
			return
		}
		*photo = File{Id: bson.NewObjectId(), Fid: fid, Time: time.Now(), User: t.Id, Type: format}
		go db.AddFile(photo)
	}

	// resize image and upload to weedfs
	go func() {
		defer wg.Done()
		resized, err := im.Resize(int(width*ratio), int(height*ratio), magick.FBox)
		if err != nil {
			failed = true
			return
		}
		go upload(resized, &purlWebp, &photoWebp, WEBP, WEBP_FORMAT)
		go upload(resized, &purlJpeg, &photoJpeg, JPEG, JPEG_FORMAT)
	}()

	// make thumbnail and upload to weedfs
	go func() {
		defer wg.Done()
		thumbnail, err := im.CropToRatio(1.0, magick.CSCenter)
		if err != nil {
			failed = true
			return
		}
		thumbnail, err = thumbnail.Resize(THUMB_SIZE, THUMB_SIZE, magick.FBox)
		if err != nil {
			failed = true
			return
		}
		go upload(thumbnail, &thumbPurlWebp, &thumbWebp, WEBP, WEBP_FORMAT)
		go upload(thumbnail, &thumbPurlJpeg, &thumbJpeg, JPEG, JPEG_FORMAT)
	}()
	wg.Wait()

	if failed {
		return nil, errors.New("failed")
	}

	photo, err := db.AddPhoto(t.Id, photoJpeg, photoWebp, BLANK)
	photo.ImageUrl = purlJpeg

	if err != nil {
		return nil, err
	}

	if bool(webpAccept) {
		photo.ImageUrl = purlWebp
	}

	return photo, err
}

func UploadPhotoToAlbum(r *http.Request, t *Token, realtime RealtimeInterface, db UserDB, albumId bson.ObjectId, webpAccept WebpAccept) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept)
	if err != nil {
		return Render(ErrorBackend)
	}
	err = db.AddPhotoToAlbum(t.Id, albumId, photo.Id)

	if err != nil {
		return Render(ErrorBackend)
	}

	return Render(photo)
}

func UploadPhoto(r *http.Request, t *Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(photo)
}
func AddStatus(db UserDB, id bson.ObjectId, r *http.Request, t *Token) (int, []byte) {
	status := &StatusUpdate{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(status)

	if err != nil {
		return Render(ErrorBadRequest)
	}

	status, err = db.AddStatus(t.Id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func AddAlbum(db UserDB, t *Token, decoder *json.Decoder) (int, []byte) {
	album := &Album{}
	if err := decoder.Decode(album); err != nil {
		log.Println(err)
		return Render(ErrorBadRequest)
	}

	album, err := db.AddAlbum(t.Id, album)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}

	return Render(album)
}

func GetStatus(db UserDB, t *Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetStatus(id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func RemoveStatus(db UserDB, t *Token, id bson.ObjectId) (int, []byte) {
	err := db.RemoveStatusSecure(t.Id, id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render("ok")
}

func GetCurrentStatus(db UserDB, t *Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetCurrentStatus(id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func UpdateStatus(db UserDB, id bson.ObjectId, r *http.Request, t *Token) (int, []byte) {
	status := &StatusUpdate{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(status)

	if err != nil {
		return Render(ErrorBadRequest)
	}

	status, err = db.UpdateStatusSecure(t.Id, id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

// func GetStatuses(db UserDB, token TokenInterface, uid IdInterface) (int, []byte) {

// }
