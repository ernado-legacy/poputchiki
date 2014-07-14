package models

import (
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

type TestStruct struct {
	Hello string `json:"hello"`
	World int    `json:"world"`
}

func TestParser(t *testing.T) {
	Convey("Parser", t, func() {
		Convey("json", func() {
			body := `{"hello": "world", "world": 1}`
			req, _ := http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewBufferString(body)))
			req.Header.Set(ContentTypeHeader, "application/json")
			v := &TestStruct{}
			err := Parse(req, v)
			So(err, ShouldBeNil)
			So(v.Hello, ShouldEqual, "world")
			So(v.World, ShouldEqual, 1)
		})
		Convey("url", func() {
			req, _ := http.NewRequest("GET", "/?hello=world&world=1", nil)
			v := &TestStruct{}
			err := Parse(req, v)
			So(err, ShouldBeNil)
			So(v.Hello, ShouldEqual, "world")
			So(v.World, ShouldEqual, 1)
		})

		Convey("form", func() {
			req, _ := http.NewRequest("POST", "/?hello=world&world=1", nil)
			req.Form = url.Values{}
			req.Form.Add("hello", "world")
			req.Form.Add("world", "1")
			v := &TestStruct{}
			err := Parse(req, v)
			So(err, ShouldBeNil)
			So(v.Hello, ShouldEqual, "world")
			So(v.World, ShouldEqual, 1)
		})
	})
}
