package main

import (
	"bufio"
	"encoding/json"
	"github.com/ginuerzh/weedo"
	"github.com/rainycape/magick"
	// "github.com/go-martini/martini"
	"bytes"
	"io"
	"labix.org/v2/mgo/bson"
	"log"
	"os"
	"os/exec"
	"strings"
	// "strings"
	// "mime/multipart"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	THUMB_SIZE               = 200
	PHOTO_MAX_SIZE           = 1000
	PHOTO_MAX_MEGABYTES      = 20
	VIDEO_MAX_MEGABYTES      = 50
	VIDEO_MAX_LENGTH_SECONDS = 360
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
)

func GetUser(db UserDB, t *Token, id bson.ObjectId, webp WebpAccept) (int, []byte) {
	c := weedo.NewClient(weedHost, weedPort)
	user := db.Get(id)
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
	user.SetAvatarUrl(c, db, webp)
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

func Logout(db UserDB, r *http.Request, tokens TokenStorage, t *Token) (int, []byte) {
	if err := tokens.Remove(t); err != nil {
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

	if err := db.Add(u); err != nil {
		log.Println(err)
		return Render(ErrorBadRequest) // todo: change error name
	}

	t, err := tokens.Generate(u)
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

	userUpdated := &User{}
	decoder.Decode(userUpdated)

	userUpdated.Id = user.Id
	userUpdated.Balance = user.Balance
	userUpdated.Password = user.Password
	userUpdated.LastAction = user.LastAction

	err := db.Update(userUpdated)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(user)
}

func Must(err error) {
	if err != nil {
		log.Println(err)
	}
}

func SendMessage(db UserDB, destination bson.ObjectId, r *http.Request, t *Token, realtime RealtimeInterface) (int, []byte) {
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
		Must(db.RemoveMessage(id))
	}()
	return Render("message removed")
}

func GetMessagesFromUser(db UserDB, origin bson.ObjectId, r *http.Request, t *Token) (int, []byte) {
	messages, err := db.GetMessagesFromUser(t.Id, origin)
	if err != nil {
		return Render(ErrorBackend)
	}
	if messages == nil {
		return Render(ErrorUserNotFound)
	}
	return Render(messages)
}

func UploadVideo(r *http.Request, t *Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept, videoAccept VideoAccept) (int, []byte) {
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

	// parse duration
	testOutput := string(b.Bytes())
	durationToken := "Duration:"
	startDuration := strings.Index(testOutput, durationToken)
	endDuration := strings.Index(testOutput, ", start:")
	duration := strings.TrimSpace(testOutput[startDuration+len(durationToken) : endDuration])
	var hours, minutes, seconds, ms int64
	_, err = fmt.Sscanf(duration, "%02d:%02d:%02d.%02d", &hours, &minutes, &seconds, &ms)
	totalSeconds := hours*60*60 + minutes*60 + seconds
	video.Duration = totalSeconds

	// check duration
	if duration == "N/A" || totalSeconds > VIDEO_MAX_LENGTH_SECONDS {
		os.Remove(sourceFilename)
		return Render(ErrorBadRequest)
	}

	sourceFile, err = os.OpenFile(sourceFilename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println(err)
		return Render(ErrorBackend)
	}
	written, err := io.Copy(sourceFile, f)
	log.Println(written, err)
	sourceFile.Close()
	// decoding goroutine
	wg := sync.WaitGroup{}
	wg.Add(3)

	uploadReaderMp4, uploadWriterMp4 := io.Pipe()
	uploadReaderWebm, uploadWriterWebm := io.Pipe()

	progressReaderMp4, progressWriterMp4 := io.Pipe()
	progressReaderWebm, progressWriterWebm := io.Pipe()

	//  ffmpeg -i input.flv -ss 00:00:14.435 -f image2 -vframes 1 out.png
	go func() {
		c := weedo.NewClient(weedHost, weedPort)
		log.Println("making screenshot")
		defer wg.Done()
		b := bytes.NewBuffer(nil)
		file, err := os.OpenFile(sourceFilename, os.O_RDONLY, 0644)
		if err != nil {
			log.Println(err)
			return
		}
		cmd := exec.Command("/bin/bash", "-c", "ffmpeg -i - -ss 00:00:01.00 -f image2 -vframes 1 -vcodec png -")
		cmd.Stderr = os.Stderr
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
		fid, purlWebp, size, err := uploadImageToWeed(c, thumbnail, "webp")
		video.ThumbnailWebp = fid
		log.Println(fid, purlWebp, size, err)
		fid, purl, size, err := uploadImageToWeed(c, thumbnail, "jpeg")
		video.ThumbnailJpeg = fid
		video.ThumbnailUrl = purl
		if webpAccept {
			video.ThumbnailUrl = purlWebp
		}
		log.Println(fid, purl, size, err)
	}()

	go func() {
		log.Println("mp4 convert started")
		defer wg.Done()
		defer uploadWriterMp4.Close()
		// defer progressWriterMp4.Close()
		// defer uploadWriterMp4.Close()
		filename := id.Hex() + ".mp4"
		path := "/tmp/" + filename
		file, err := os.OpenFile(sourceFilename, os.O_RDONLY, 0644)
		decodeReader := io.TeeReader(file, progressWriterMp4)
		defer progressWriterWebm.Close()
		if err != nil {
			log.Println(err)
			return
		}
		defer os.Remove(path)
		cmd := exec.Command("/bin/bash", "-c", "ffmpeg -i - -vcodec h264 -acodec aac -strict -2 -vf scale=300:ih*300/iw,crop=out_w=in_h  "+path)
		// cmd.Stderr = os.Stdout
		cmd.Stdin = decodeReader

		if e := cmd.Start(); e != nil {
			log.Println(e)
			return
		}
		cmd.Wait()
		log.Println("mp4 ok")
		file.Close()
		file, err = os.Open(path)
		if err != nil {
			log.Println(err)
			return
		}

		io.Copy(uploadWriterMp4, file)
		file.Close()
	}()

	go func() {
		log.Println("mp4 convert started")
		defer wg.Done()
		defer uploadWriterWebm.Close()
		// defer progressWriterMp4.Close()
		// defer uploadWriterMp4.Close()
		filename := id.Hex() + ".webm"
		path := "/tmp/" + filename
		file, err := os.OpenFile(sourceFilename, os.O_RDONLY, 0644)
		if err != nil {
			log.Println(err)
			return
		}
		decodeReader := io.TeeReader(file, progressWriterWebm)
		defer progressWriterWebm.Close()
		defer os.Remove(path)
		cmd := exec.Command("/bin/bash", "-c", "ffmpeg -i - -c:v libvpx -b:v 256k -c:a libvorbis -cpu-used 4 -vf scale=300:ih*300/iw,crop=out_w=in_h "+path)
		// cmd.Stderr = os.Stdout
		cmd.Stdin = decodeReader

		if e := cmd.Start(); e != nil {
			log.Println(e)
			return
		}
		cmd.Wait()
		log.Println("webm ok")
		file.Close()
		file, err = os.Open(path)
		if err != nil {
			log.Println(err)
			return
		}

		io.Copy(uploadWriterWebm, file)
		file.Close()
	}()

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
		os.Remove(sourceFilename)
	}()

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
		log.Println("progress hahdling finished")
	}()

	go func() {
		c := weedo.NewClient(weedHost, weedPort)
		wg := sync.WaitGroup{}
		wg.Add(2)
		fileWebm := File{}
		fileMp4 := File{}
		var ok error
		ok = nil
		go func() {
			defer wg.Done()
			fid, purl, size, err := uploadToWeed(c, uploadReaderWebm, "video", "webm")
			if err != nil {
				ok = err
			}
			fileWebm = File{bson.NewObjectId(), fid, t.Id, time.Now(), "video/webm", size, purl}
		}()
		go func() {
			defer wg.Done()
			fid, purl, size, err := uploadToWeed(c, uploadReaderMp4, "video", "mp4")
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
			video.VideoUrl = fileMp4.Url
		}
		if videoAccept == VA_WEBM {
			video.VideoUrl = fileWebm.Url
		}
		realtime.Push(t.Id, video)
	}()
	_, err = db.AddVideo(&video)
	if err != nil {
		return Render(ErrorBackend)
	}
	return Render(video)
}

// reads data from io.Reader, uploads it with type/format and returs fid, purl and error
func uploadToWeed(c *weedo.Client, reader io.Reader, t, format string) (string, string, int64, error) {
	fid, size, err := c.AssignUpload(t+"."+format, t+"/"+format, reader)
	if err != nil {
		log.Println(t, format, err)
		return "", "", size, err
	}
	purl, _, err := c.GetUrl(fid)
	if err != nil {
		log.Println(err)
		return "", "", size, err
	}
	log.Println(t, format, "uploaded", purl)
	return fid, purl, size, nil
}

func uploadImageToWeed(c *weedo.Client, image *magick.Image, format string) (string, string, int64, error) {
	encodeReader, encodeWriter := io.Pipe()
	go func() {
		defer encodeWriter.Close()
		info := magick.NewInfo()
		info.SetFormat(format)
		if err := image.Encode(encodeWriter, info); err != nil {
			log.Println(err)
		}
	}()
	return uploadToWeed(c, encodeReader, "image", format)
}

func pushProgress(length int64, rate int64, progressWriter *io.PipeWriter, progressReader *io.PipeReader, realtime RealtimeInterface, t *Token) {
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

func uploadPhoto(r *http.Request, t *Token, realtime RealtimeInterface, db UserDB, webpAccept WebpAccept) (*Photo, error) {
	c := weedo.NewClient(weedHost, weedPort)
	f, _, err := r.FormFile(FORM_FILE)
	if err != nil {
		log.Println("unable to read form file", err)
		return nil, err
	}

	length := r.ContentLength
	if length > 1024*1024*PHOTO_MAX_MEGABYTES {
		return nil, errors.New("bad request")
	}

	progressReader, progressWriter := io.Pipe()
	decodeReader := io.TeeReader(f, progressWriter)

	// download progress goroutine
	go pushProgress(length, 10, progressWriter, progressReader, realtime, t)

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
		fid, purl, size, err := uploadImageToWeed(c, image, extension)
		*url = purl
		if err != nil {
			failed = true
			return
		}
		*photo = File{Id: bson.NewObjectId(), Fid: fid, Time: time.Now(), User: t.Id, Type: format, Size: size}
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

	photo, err := db.AddPhoto(t.Id, photoJpeg, photoWebp, thumbJpeg, thumbWebp, BLANK)
	photo.ImageUrl = purlJpeg
	photo.ThumbnailUrl = thumbPurlJpeg

	if err != nil {
		return nil, err
	}

	if bool(webpAccept) {
		photo.ImageUrl = purlWebp
		photo.ThumbnailUrl = thumbPurlWebp
	}

	return photo, err
}

func UploadPhotoToAlbum(r *http.Request, t *Token, realtime RealtimeInterface, db UserDB, albumId bson.ObjectId, webpAccept WebpAccept) (int, []byte) {
	photo, err := uploadPhoto(r, t, realtime, db, webpAccept)
	if err != nil {
		return Render(ErrorBackend)
	}
	if db.AddPhotoToAlbum(t.Id, albumId, photo.Id) != nil {
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
	if err := decoder.Decode(status); err != nil {
		return Render(ErrorBadRequest)
	}
	status, err := db.AddStatus(t.Id, status.Text)
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
	if err := db.RemoveStatusSecure(t.Id, id); err != nil {
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

func UpdateStatus(db UserDB, id bson.ObjectId, r *http.Request, t *Token, decoder *json.Decoder) (int, []byte) {
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
