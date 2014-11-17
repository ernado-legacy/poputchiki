package main

import (
	"flag"
	"fmt"
	"github.com/ernado/poputchiki/database"
	"github.com/ernado/poputchiki/models"
	"github.com/ernado/selectel/storage"
	"github.com/ernado/weed"
	"gopkg.in/mgo.v2"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type Data struct {
	Content string
	Fid     string
}

var (
	dbName        = "poputchiki-test"
	dbHost        = "localhost:27018"
	dbSalt        = "salt"
	weedUrl       = "http://vpn.cydev.ru:9333"
	containerName = "poputchiki"
	db            *database.DB
	adapter       *weed.Adapter
	selectel      storage.API
	threads       = 15
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func process() {
	container := selectel.Container(containerName)
	wg := new(sync.WaitGroup)
	objects, err := container.ObjectsInfo()
	must(err)
	files := make(map[string]bool)
	for _, object := range objects {
		files[object.Name] = true
	}
	log.Println("Objects in storage:", len(files))
	dataChan := make(chan Data, 50)
	worker := func(c chan Data) {
		wg.Add(1)
		log.Println("worker started")
		defer wg.Done()
		defer log.Println("worker stopped")
		for data := range c {
			if files[data.Fid] {
				fmt.Printf("file %s already in storage\n", data.Fid)
				continue
			}
			if len(data.Fid) == 0 {
				fmt.Printf("bad length\n")
				continue
			}
			fmt.Printf("Uploading %s %s\n", data.Content, data.Fid)
			url, err := adapter.GetUrl(data.Fid)
			if err != nil {
				log.Println(data.Fid, err)
				continue
			}
			res, err := http.Get(url)
			if err != nil {
				log.Println(data.Fid, err)
				continue
			}
			if err := container.Upload(res.Body, data.Fid, data.Content); err != nil {
				log.Println("error", err, data.Fid)
				continue
			}
			fmt.Printf("Uploaded %s %s\n", data.Content, data.Fid)
		}
	}
	for i := 0; i < threads; i++ {
		go worker(dataChan)
	}
	photos, _, err := db.SearchAllPhoto(models.Pagination{Count: 10000})
	must(err)
	for _, photo := range photos {
		dataChan <- Data{"image/jpeg", photo.ImageJpeg}
		dataChan <- Data{"image/webp", photo.ImageWebp}
		dataChan <- Data{"image/jpeg", photo.ThumbnailJpeg}
		dataChan <- Data{"image/webp", photo.ThumbnailWebp}
	}
	audio, err := db.GetAllAudio()
	must(err)
	for _, a := range audio {
		dataChan <- Data{"audio/mp3", a.AudioAac}
		dataChan <- Data{"audio/ogg", a.AudioOgg}
	}
	video, err := db.GetAllVideo()
	must(err)
	for _, v := range video {
		dataChan <- Data{"video/mp4", v.VideoMpeg}
		dataChan <- Data{"video/webm", v.VideoWebm}
		dataChan <- Data{"image/jpeg", v.ThumbnailJpeg}
		dataChan <- Data{"image/webp", v.ThumbnailWebp}
	}
	close(dataChan)
	wg.Wait()
}

func main() {
	flag.StringVar(&dbName, "db.name", dbName, "Database name")
	flag.StringVar(&dbHost, "db.host", dbHost, "Mongo host")
	flag.StringVar(&dbSalt, "db.salt", dbSalt, "Database salt")
	flag.StringVar(&weedUrl, "weed.url", weedUrl, "WeedFs url")
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("connecting to db...")
	session, err := mgo.Dial(dbHost)
	must(err)
	log.Println("connecting to storage...")
	selectel, err = storage.NewEnv()
	must(err)
	db = database.New(dbName, dbSalt, time.Second, session)
	adapter = weed.NewAdapter(weedUrl)
	process()
}
