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

type TestStructStr struct {
	Hello string `json:"hello"`
	World string `json:"world"`
}

type TestStructList struct {
	World []string `json:"world"`
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

		Convey("String reflection", func() {
			req, _ := http.NewRequest("GET", "/?hello=world&world=1", nil)
			v := &TestStructStr{}
			err := Parse(req, v)
			So(err, ShouldBeNil)
			So(v.Hello, ShouldEqual, "world")
			So(v.World, ShouldEqual, "1")
		})

		Convey("form", func() {
			req, _ := http.NewRequest("POST", "/", nil)
			req.PostForm = url.Values{}
			req.PostForm.Add("hello", "world")
			req.PostForm.Add("world", "1")
			req.Header.Set(ContentTypeHeader, "x-www-form-urlencoded")
			v := &TestStruct{}
			err := Parse(req, v)
			So(err, ShouldBeNil)
			So(v.Hello, ShouldEqual, "world")
			So(v.World, ShouldEqual, 1)
		})
		Convey("arrays", func() {
			Convey("String to array", func() {
				req, _ := http.NewRequest("GET", "/?world=world", nil)
				v := &TestStructList{}
				err := Parse(req, v)
				So(err, ShouldBeNil)
				So(len(v.World), ShouldEqual, 1)
				So(v.World[0], ShouldEqual, "world")
			})
			Convey("Array to array", func() {
				req, _ := http.NewRequest("GET", "/?world=world&world=hello", nil)
				v := &TestStructList{}
				err := Parse(req, v)
				So(err, ShouldBeNil)
				So(len(v.World), ShouldEqual, 2)
				So(v.World[0], ShouldEqual, "world")
				So(v.World[1], ShouldEqual, "hello")
			})
		})
	})
}
