package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestWeedUrl(t *testing.T) {
	Convey("Url ok", t, func() {
		fid := "5,0a87c48712af"
		w := NewAdapter()
		url, err := w.GetUrl(fid)
		So(err, ShouldBeNil)
		So(url, ShouldEqual, "msk1.cydev.ru:8080/5,0a87c48712af")
		Convey("Caching enabled", func() {
			urlChan := make(chan string)
			errChan := make(chan error)
			go func() {
				url, err := w.GetUrl(fid)
				if err != nil {
					errChan <- err
				}
				urlChan <- url
			}()
			select {
			case <-time.After(100 * time.Nanosecond):
				So(true, ShouldBeFalse)
			case url := <-urlChan:
				So(url, ShouldEqual, "msk1.cydev.ru:8080/5,0a87c48712af")
			case <-errChan:
				So(true, ShouldBeFalse)
			}
		})
	})
}
