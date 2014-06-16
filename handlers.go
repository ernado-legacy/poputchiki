package main

import unsafe "unsafe"

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ernado/cymedia/photo"
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki-api/weed"
	"github.com/rainycape/magick"
	"io"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"os"
	"os/exec"
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

func GetUser(db UserDB, t *gotok.Token, id bson.ObjectId, webp WebpAccept, adapter *weed.Adapter) (int, []byte) {
	log.Println("TEST")
	userChannel := make(chan *User)
	go func() {
		userChannel <- db.Get(id)
	}()

	var user *User
	const infoSize = unsafe.Sizeof(User{})
	log.Println(infoSize)

	select {
	case <-time.After(time.Millisecond * 10):
		log.Println("database get timed out")
		return Render(ErrorBackend)
	case u := <-userChannel:
		user = u
	}

	if user == nil {
		return Render(ErrorUserNotFound)
	}
	for _, u := range user.Blacklist {
		if u == t.Id {
			return Render(ErrorBlacklisted)
		}
	}
	// hiding private fields for non-owner
	if t == nil || t.Id != id {
		user.CleanPrivate()
	}

	ok := make(chan time.Time)
	go func() {
		user.Prepare(adapter, db, webp)
		ok <- time.Now()
	}()
	select {
	case <-time.After(time.Millisecond * 70):
		log.Println("prepare")
	case <-ok:
		return Render(user)
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
	if err := db.AddToFavorites(user.Id, friend.Id); err != nil {
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
	if err := db.AddToBlacklist(user.Id, friend.Id); err != nil {
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
	if err := db.RemoveFromBlacklist(user.Id, friend.Id); err != nil {
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
	if err := db.RemoveFromFavorites(user.Id, friend.Id); err != nil {
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
		return Render(ErrorUserNotFound) // todo: rename error
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

func Login(db UserDB, r *http.Request, tokens gotok.Storage) (int, []byte) {
	username, password := r.FormValue(FORM_EMAIL), r.FormValue(FORM_PASSWORD)
	user := db.GetUsername(username)
	if user == nil {
		return Render(ErrorUserNotFound)
	}
	if user.Password != getHash(password) {
		return Render(ErrorAuth)
	}
	t, err := tokens.Generate(user.Id)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(t)
}

func Logout(db UserDB, r *http.Request, tokens gotok.Storage, t *gotok.Token) (int, []byte) {
	if err := tokens.Remove(t); err != nil {
		return Render(ErrorBackend)
	}
	return Render("logged out")
}

func Register(db UserDB, r *http.Request, tokens gotok.Storage) (int, []byte) {
	u := UserFromForm(r)
	uDb := db.GetUsername(u.Email)
	if uDb != nil {
		return Render(ErrorBadRequest) // todo: change error name
	}

	if err := db.Add(u); err != nil {
		log.Println(err)
		return Render(ErrorBadRequest) // todo: change error name
	}

	t, err := tokens.Generate(u.Id)
	if err != nil {
		return Render(ErrorBackend)
	}

	return Render(t)
}

func Update(db UserDB, r *http.Request, id bson.ObjectId, decoder *json.Decoder) (int, []byte) {
	user := db.Get(id)
	if user == nil {
		return Render(ErrorUserNotFound)
	}

	e := decoder.Decode(user)
	if e != nil {
		log.Println(e)
		return Render(ErrorBadRequest)
	}
	user.Id = user.Id
	user.Balance = user.Balance
	user.Password = user.Password
	user.LastAction = user.LastAction

	err := db.Update(user)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	return Render(user)
}

func Must(err error) {
	if err != nil {
		log.Println(err)
	}
}

func SendMessage(db UserDB, destination bson.ObjectId, r *http.Request, t *gotok.Token, realtime RealtimeInterface) (int, []byte) {
	text := r.FormValue(FORM_TEXT)
	origin := t.Id
	now := time.Now()

	if text == BLANK {
		return Render(ErrorBadRequest)
	}

	m1 := Message{bson.NewObjectId(), origin, origin, destination, now, text}
	m2 := Message{bson.NewObjectId(), destination, origin, destination, now, text}

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

func RemoveMessage(db UserDB, id bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
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

func GetMessagesFromUser(db UserDB, origin bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
	messages, err := db.GetMessagesFromUser(t.Id, origin)
	if err != nil {
		return Render(ErrorBackend)
	}
	if messages == nil {
		return Render(ErrorUserNotFound)
	}
	return Render(messages)
}

func UploadVideo(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept, videoAccept VideoAccept, adapter *weed.Adapter) (int, []byte) {
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
		if videoAccept == VA_MP4 {
			video.VideoUrl = fileMp4.Url + ".mp4"
		}
		if videoAccept == VA_WEBM {
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

func uploadPhoto(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept, adapter *weed.Adapter) (*Photo, error) {
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read form file", err)
		return nil, err
	}

	length := r.ContentLength
	if length > 1024*1024*PHOTO_MAX_MEGABYTES {
		return nil, errors.New("bad request")
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

	newPhoto, err := db.AddPhoto(t.Id, c(&p.ImageJpeg), c(&p.ImageWebp), c(&p.ThumbnailJpeg), c(&p.ThumbnailWebp), BLANK)
	newPhoto.ImageUrl = p.ImageJpeg.Url
	newPhoto.ThumbnailUrl = p.ThumbnailJpeg.Url

	if bool(webpAccept) {
		newPhoto.ImageUrl = p.ImageWebp.Url
		newPhoto.ThumbnailUrl = p.ThumbnailWebp.Url
	}
	return newPhoto, err
}

func UploadPhotoToAlbum(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db UserDB, albumId bson.ObjectId, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept, adapter)
	if err != nil {
		return Render(ErrorBackend)
	}
	if db.AddPhotoToAlbum(t.Id, albumId, photo.Id) != nil {
		return Render(ErrorBackend)
	}
	return Render(photo)
}

func UploadPhoto(r *http.Request, t *gotok.Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept, adapter)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(photo)
}

func AddStatus(db UserDB, id bson.ObjectId, r *http.Request, t *gotok.Token) (int, []byte) {
	status := &StatusUpdate{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(status); err != nil {
		return Render(ErrorBadRequest)
	}
	status, err := db.AddStatus(t.Id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func AddAlbum(db UserDB, t *gotok.Token, decoder *json.Decoder) (int, []byte) {
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

func GetStatus(db UserDB, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetStatus(id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func RemoveStatus(db UserDB, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	if err := db.RemoveStatusSecure(t.Id, id); err != nil {
		return Render(ErrorBackend)
	}
	return Render("ok")
}

func GetCurrentStatus(db UserDB, t *gotok.Token, id bson.ObjectId) (int, []byte) {
	status, err := db.GetCurrentStatus(id)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func UpdateStatus(db UserDB, id bson.ObjectId, r *http.Request, t *gotok.Token, decoder *json.Decoder) (int, []byte) {
	status := &StatusUpdate{}
	if err := decoder.Decode(status); err != nil {
		return Render(ErrorBadRequest)
	}
	status, err := db.UpdateStatusSecure(t.Id, id, status.Text)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(status)
}

func SearchPeople(db UserDB, pagination Pagination, r *http.Request, webpAccept WebpAccept, adapter *weed.Adapter) (int, []byte) {
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

	log.Println(query)
	for key, _ := range result {
		result[key].Prepare(adapter, db, webpAccept)
	}

	return Render(result)
}

func AddStripeItem(db UserDB, t *gotok.Token, decoder *json.Decoder) (int, []byte) {
	var media interface{}
	request := &StripeItemRequest{}
	if decoder.Decode(request) != nil {
		return Render(ErrorBadRequest)
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
