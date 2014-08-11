package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ernado/gotok"
	. "github.com/ernado/poputchiki/models"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStripeUpdate(t *testing.T) {
	username := "test@" + mailDomain
	password := "secretsecret"
	*development = true
	redisName = "poputchiki_test_upload"
	dbName = "poputchiki_dev_upload"
	path := "test/image.jpg"
	a := NewApp()
	defer a.Close()
	a.DropDatabase()

	Convey("Registration with unique username and valid password should be successfull", t, func() {
		Reset(a.DropDatabase)
		token := new(gotok.Token)
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
		a.ServeHTTP(res, req)
		So(res.Code, ShouldEqual, http.StatusOK)
		decoder := json.NewDecoder(res.Body)
		So(decoder.Decode(token), ShouldBeNil)
		Convey("Request should completed", func() {
			file, err := os.Open(path)
			So(err, ShouldBeNil)
			res := httptest.NewRecorder()
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", filepath.Base(path))
			So(err, ShouldBeNil)
			_, err = io.Copy(part, file)
			So(err, ShouldBeNil)
			So(writer.Close(), ShouldBeNil)
			req, err := http.NewRequest("POST", "/api/photo/?token="+token.Token, body)
			So(err, ShouldBeNil)
			req.Header.Add("Content-type", writer.FormDataContentType())
			a.ServeHTTP(res, req)
			So(res.Code, ShouldEqual, http.StatusOK)
			imageBody, _ := ioutil.ReadAll(res.Body)
			image := &Photo{}
			log.Println(string(imageBody))
			So(json.Unmarshal(imageBody, image), ShouldBeNil)

			Convey("File must be able to download", func() {
				req, _ = http.NewRequest("GET", image.ImageUrl, nil)
				client := &http.Client{}
				res, err := client.Do(req)
				So(err, ShouldBeNil)
				So(res.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Stipe add", func() {
				sreq := &StripeItemRequest{image.Id, "photo"}
				j, err := json.Marshal(sreq)
				So(err, ShouldBeNil)
				res := httptest.NewRecorder()
				body := bytes.NewBuffer(j)
				req, err := http.NewRequest("POST", "/api/stripe/?token="+token.Token, body)
				So(err, ShouldBeNil)
				req.Header.Add("Content-type", "application/json")
				a.ServeHTTP(res, req)
				So(res.Code, ShouldEqual, http.StatusOK)
				Convey("Stripe get", func() {
					res := httptest.NewRecorder()
					req, err := http.NewRequest("GET", "/api/stripe?token="+token.Token, nil)
					So(err, ShouldBeNil)
					a.ServeHTTP(res, req)
					So(res.Code, ShouldEqual, http.StatusOK)
					stripe := []StripeItem{}
					decoder := json.NewDecoder(res.Body)
					So(decoder.Decode(&stripe), ShouldBeNil)
					So(len(stripe), ShouldEqual, 1)
					s := stripe[0]
					So(s.Type, ShouldEqual, "photo")
					So(s.ImageUrl, ShouldNotEqual, "")
				})
			})

		})
	})
}
func TestStatusUpdate(t *testing.T) {
	username := "test@" + mailDomain
	password := "secretsecret"
	*development = true
	redisName = "poputchiki_test_status"
	dbName = "poputchiki_test_status"
	a := NewApp()
	defer a.Close()
	a.DropDatabase()

	Convey("Registration with unique username and valid password should be successfull", t, func() {
		Reset(func() {
			a.DropDatabase()
		})

		res := httptest.NewRecorder()
		// sending registration request
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
		a.ServeHTTP(res, req)

		// reading response
		So(res.Code, ShouldEqual, http.StatusOK)
		tokenBody, _ := ioutil.ReadAll(res.Body)
		token := &gotok.Token{}
		So(json.Unmarshal(tokenBody, token), ShouldBeNil)

		Convey("Status set", func() {
			text := "hello"
			res := httptest.NewRecorder()
			url := fmt.Sprintf("/api/status?text=%s&token=%s", text, token.Token)
			req, _ := http.NewRequest("POST", url, nil)
			a.ServeHTTP(res, req)
			So(res.Code, ShouldEqual, http.StatusOK)
			Convey("Status get", func() {
				res := httptest.NewRecorder()
				s := new(StatusUpdate)
				url := fmt.Sprintf("/api/user/%s/status?token=%s", token.Id.Hex(), token.Token)
				req, _ := http.NewRequest("GET", url, nil)
				a.ServeHTTP(res, req)
				decoder := json.NewDecoder(res.Body)
				So(res.Code, ShouldEqual, http.StatusOK)
				So(decoder.Decode(s), ShouldBeNil)
				So(s.Text, ShouldEqual, text)
			})

			Convey("User get", func() {
				res := httptest.NewRecorder()
				s := new(User)
				url := fmt.Sprintf("/api/user/%s?token=%s", token.Id.Hex(), token.Token)
				req, _ := http.NewRequest("GET", url, nil)
				a.ServeHTTP(res, req)
				decoder := json.NewDecoder(res.Body)
				So(res.Code, ShouldEqual, http.StatusOK)
				So(decoder.Decode(s), ShouldBeNil)
				So(s.Status, ShouldEqual, text)
			})

			Convey("Second status set should fail", func() {
				text := "hello"
				res := httptest.NewRecorder()
				url := fmt.Sprintf("/api/status?text=%s&token=%s", text, token.Token)
				req, _ := http.NewRequest("POST", url, nil)
				a.ServeHTTP(res, req)
				So(res.Code, ShouldEqual, http.StatusPaymentRequired)

				Convey("So user should activate vip", func() {
					res := httptest.NewRecorder()
					url := fmt.Sprintf("/api/vip/week?token=%s", token.Token)
					req, _ := http.NewRequest("POST", url, nil)
					a.ServeHTTP(res, req)
					So(res.Code, ShouldEqual, http.StatusOK)

					Convey("And add 2 statuses", func() {
						text := "hello"
						res := httptest.NewRecorder()
						url := fmt.Sprintf("/api/status?text=%s&token=%s", text, token.Token)
						req, _ := http.NewRequest("POST", url, nil)
						a.ServeHTTP(res, req)
						So(res.Code, ShouldEqual, http.StatusOK)

						res = httptest.NewRecorder()
						a.ServeHTTP(res, req)
						So(res.Code, ShouldEqual, http.StatusOK)

						Convey("Second status set should fail", func() {
							res = httptest.NewRecorder()
							a.ServeHTTP(res, req)
							So(res.Code, ShouldEqual, http.StatusPaymentRequired)
						})
					})
				})
			})
		})
	})
}

func TestUpload(t *testing.T) {
	username := "test@" + mailDomain
	password := "secretsecret"
	*development = true
	redisName = "poputchiki_test_upload"
	dbName = "poputchiki_dev_upload"
	path := "test/image.jpg"
	a := NewApp()
	defer a.Close()
	a.DropDatabase()

	Convey("Registration with unique username and valid password should be successfull", t, func() {
		Reset(func() {
			a.DropDatabase()
		})

		res := httptest.NewRecorder()
		// sending registration request
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
		a.ServeHTTP(res, req)

		// reading response
		So(res.Code, ShouldEqual, http.StatusOK)
		tokenBody, _ := ioutil.ReadAll(res.Body)
		token := &gotok.Token{}
		So(json.Unmarshal(tokenBody, token), ShouldBeNil)

		Convey("Request should completed", func() {
			file, err := os.Open(path)
			So(err, ShouldBeNil)
			res := httptest.NewRecorder()
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", filepath.Base(path))
			So(err, ShouldBeNil)
			_, err = io.Copy(part, file)
			So(err, ShouldBeNil)
			So(writer.Close(), ShouldBeNil)
			req, err := http.NewRequest("POST", "/api/photo/?token="+token.Token, body)
			So(err, ShouldBeNil)
			req.Header.Add("Content-type", writer.FormDataContentType())
			a.ServeHTTP(res, req)
			So(res.Code, ShouldEqual, http.StatusOK)
			imageBody, _ := ioutil.ReadAll(res.Body)
			image := &Photo{}
			log.Println(string(imageBody))
			So(json.Unmarshal(imageBody, image), ShouldBeNil)

			Convey("File must be able to download", func() {
				req, _ = http.NewRequest("GET", image.ImageUrl, nil)
				client := &http.Client{}
				res, err := client.Do(req)
				So(err, ShouldBeNil)
				So(res.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Stipe", func() {
				sreq := &StripeItemRequest{image.Id, "photo"}
				j, err := json.Marshal(sreq)
				So(err, ShouldBeNil)
				res := httptest.NewRecorder()
				body := bytes.NewBuffer(j)
				req, err := http.NewRequest("POST", "/api/stripe/?token="+token.Token, body)
				So(err, ShouldBeNil)
				req.Header.Add("Content-type", "application/json")

				a.ServeHTTP(res, req)
				So(res.Code, ShouldEqual, http.StatusOK)
			})

		})
	})
}

func TestRealtime(t *testing.T) {
	redisName = "poputchiki_test_realtime"
	pool := newPool()
	realtime := &RealtimeRedis{pool, make(map[bson.ObjectId]ReltChannel)}
	id := bson.NewObjectId()
	event := "test"
	c := realtime.GetWSChannel(id)
	err := realtime.Push(id, event)
	eventRec := <-c.channel
	Convey("Push ok", t, func() {
		So(err, ShouldEqual, nil)
		Convey("And event should be delivered", func() {
			So(eventRec.Body, ShouldEqual, event)
			So(eventRec.Type, ShouldEqual, "string")
		})
	})
}

func TestGeoSearch(t *testing.T) {
	dbName = "poputchiki_geo"
	redisName = "poputchiki_geo"
	a := NewApp()
	username := "m@cydev.ru"
	password := "secretsecret"
	firstname := "kek"
	var tokenBody []byte
	var token1 gotok.Token

	Convey("Register", t, func() {
		Reset(func() {
			a.DropDatabase()
			a.InitDatabase()
		})

		res := httptest.NewRecorder()
		// sending registration request
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
		req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
		a.ServeHTTP(res, req)

		// reading response
		tokenBody, _ = ioutil.ReadAll(res.Body)
		So(res.Code, ShouldEqual, http.StatusOK)
		Convey("User should be able to change information after registration", func() {
			err := json.Unmarshal(tokenBody, &token1)
			So(err, ShouldEqual, nil)

			reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token1.Id.Hex(), token1.Token)
			changes := User{}
			changes.Name = firstname
			changes.Sex = "male"
			changes.Location = []float64{155.0, 155.0}
			uJson, err := json.Marshal(changes)
			uReader := bytes.NewReader(uJson)
			So(err, ShouldBeNil)
			req, _ := http.NewRequest("PATCH", reqUrl, uReader)
			req.Header.Add(ContentTypeHeader, "application/json")
			a.ServeHTTP(res, req)

			So(res.Code, ShouldEqual, http.StatusOK)
			Convey("Search", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)
				time.Sleep(500 * time.Millisecond)
				reqUrl := fmt.Sprintf("/api/search/?geo=true&token=%s", token1.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusOK)
				result := new(SearchResult)
				resultBody, _ := ioutil.ReadAll(res.Body)
				err = json.Unmarshal(resultBody, result)
				So(err, ShouldBeNil)
				So(result.Count, ShouldEqual, 1)
			})
		})

	})
}

func TestMethods(t *testing.T) {
	username := "m@cydev.ru"
	password := "secretsecret"
	firstname := "Ivan"
	secondname := "Pupkin"
	phone := "+79197241488"

	messageText := "hello world русский текст"

	dbName = "poputchiki_dev"
	*development = true
	*sendEmail = false
	redisName = "poputchiki_dev"

	a := NewApp()
	a.DropDatabase()
	a.InitDatabase()
	a.Close()

	a = NewApp()

	var tokenBody []byte
	var token1 gotok.Token

	Convey("Registration with unique username and valid password should be successfull", t, func() {

		res := httptest.NewRecorder()
		// sending registration request
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
		req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
		a.ServeHTTP(res, req)

		// reading response
		tokenBody, _ = ioutil.ReadAll(res.Body)
		So(res.Code, ShouldEqual, http.StatusOK)

		Convey("User GET error handling", func() {
			Convey("400 Bad request", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)
				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId(), token1.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusBadRequest)
			})
			Convey("401 Unauthorized", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)
				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId().Hex(), "badtoken")
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusUnauthorized)
			})
			Convey("404 Not found with nonexistent id", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)
				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId().Hex(), token1.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("User PATCH error handling", func() {
			Convey("400 Bad request", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)

				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId(), token1.Token)
				req, _ := http.NewRequest("PATCH", reqUrl, nil)
				req.PostForm = url.Values{FORM_FIRSTNAME: {firstname}, FORM_SECONDNAME: {secondname}, FORM_PHONE: {phone}}
				req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusBadRequest)
			})
			Convey("401 Unauthorized", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)

				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId().Hex(), bson.NewObjectId().Hex())
				req, _ := http.NewRequest("PATCH", reqUrl, nil)
				req.PostForm = url.Values{FORM_FIRSTNAME: {firstname}, FORM_SECONDNAME: {secondname}, FORM_PHONE: {phone}}
				req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusUnauthorized)
			})
			Convey("405 Not allowed with nonexistent id or id != token id", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)

				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId().Hex(), token1.Token)
				req, _ := http.NewRequest("PATCH", reqUrl, nil)
				req.PostForm = url.Values{FORM_FIRSTNAME: {firstname}, FORM_SECONDNAME: {secondname}, FORM_PHONE: {phone}}
				req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusMethodNotAllowed)
			})
		})

		Convey("Login error handling", func() {
			Convey("404 Not found - user is nonexistent", func() {
				res := httptest.NewRecorder()
				// trying to log in
				req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
				req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {"randomemail"}}
				req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
				a.ServeHTTP(res, req)

				So(res.Code, ShouldEqual, http.StatusNotFound)
				a.DropDatabase()
			})
			Convey("404 Unauthorised - incorrect password", func() {
				res := httptest.NewRecorder()
				// trying to log in
				req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
				req.PostForm = url.Values{FORM_PASSWORD: {"randompass"}, FORM_EMAIL: {username}}
				req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
				a.ServeHTTP(res, req)

				So(res.Code, ShouldEqual, http.StatusUnauthorized)
				a.DropDatabase()
			})
		})

		Convey("User should be able to set status", func() {
			res := httptest.NewRecorder()
			err := json.Unmarshal(tokenBody, &token1)
			So(err, ShouldEqual, nil)
			reqUrl := fmt.Sprintf("/api/status?token=%s", token1.Token)
			status := new(StatusUpdate)
			status.Text = "Hello world"
			body, err := json.Marshal(status)
			So(err, ShouldBeNil)

			req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(body))
			a.ServeHTTP(res, req)
			a.DropDatabase()
			So(res.Code, ShouldEqual, http.StatusOK)
		})

		Convey("User should be able to change information after registration", func() {
			a.InitDatabase()
			err := json.Unmarshal(tokenBody, &token1)
			So(err, ShouldEqual, nil)

			reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token1.Id.Hex(), token1.Token)
			changes := User{}
			changes.Name = firstname
			changes.Phone = phone
			changes.Sex = "female"
			changes.Location = []float64{155.0, 155.0}
			uJson, err := json.Marshal(changes)
			uReader := bytes.NewReader(uJson)
			So(err, ShouldBeNil)
			req, _ := http.NewRequest("PATCH", reqUrl, uReader)
			req.Header.Add(ContentTypeHeader, "application/json")
			a.ServeHTTP(res, req)

			So(res.Code, ShouldEqual, http.StatusOK)
			// a.DropDatabase()
			Convey("And changes must me applied", func() {
				res := httptest.NewRecorder()

				// 	// trying to get user information with scope

				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token1.Id.Hex(), token1.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)

				So(res.Code, ShouldEqual, http.StatusOK)
				u := User{}
				userBody, _ := ioutil.ReadAll(res.Body)
				json.Unmarshal(userBody, &u)
				a.DropDatabase()
				So(u.Name, ShouldEqual, changes.Name)
				So(u.Phone, ShouldEqual, changes.Phone)
				So(u.Sex, ShouldEqual, changes.Sex)
				So(len(u.Location), ShouldEqual, 2)
				So(u.Location[0], ShouldAlmostEqual, changes.Location[0])
				So(u.Location[1], ShouldAlmostEqual, changes.Location[1])
			})
			Convey("Search", func() {
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)
				a.InitDatabase()
				time.Sleep(500 * time.Millisecond)
				reqUrl := fmt.Sprintf("/api/search/?sex=female&token=%s", token1.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusOK)
				result := new(SearchResult)
				resultBody, _ := ioutil.ReadAll(res.Body)
				err = json.Unmarshal(resultBody, result)
				So(err, ShouldBeNil)
				So(result.Count, ShouldEqual, 1)
			})
		})

		Convey("User should be able to log in after registration", func() {
			res := httptest.NewRecorder()
			// trying to log in
			req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
			req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
			req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
			a.ServeHTTP(res, req)

			So(res.Code, ShouldEqual, http.StatusOK)

			Convey("Returned token must be valid json object", func() {
				// parsing json response to token object
				err := json.Unmarshal(tokenBody, &token1)
				id := token1.Id

				// simple token validation
				So(err, ShouldEqual, nil)
				So(token1.Token, ShouldNotBeBlank)
				So(token1.Id.Hex(), ShouldNotBeBlank)

				Convey("And user must be able to use it", func() {
					res := httptest.NewRecorder()

					// trying to get user information with scope
					reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", id.Hex(), token1.Token)
					req, _ := http.NewRequest("GET", reqUrl, nil)
					a.ServeHTTP(res, req)

					a.DropDatabase()
					So(res.Code, ShouldEqual, http.StatusOK)
				})
				Convey("And log out after that", func() {
					res := httptest.NewRecorder()
					// trying to log out
					reqUrl := fmt.Sprintf("/api/auth/logout/?token=%s", token1.Token)
					req, _ := http.NewRequest("POST", reqUrl, nil)
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusOK)
					Convey("And user must not be able to use deleted token anymore", func() {
						res := httptest.NewRecorder()

						// trying to get user information with scope
						reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", id.Hex(), token1.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						a.ServeHTTP(res, req)

						a.DropDatabase()
						So(res.Code, ShouldEqual, http.StatusUnauthorized)
					})
				})
			})
		})
		Convey("Returned token must be valid", func() {
			// parsing registration token
			t := gotok.Token{}
			err := json.Unmarshal(tokenBody, &t)

			// validating
			So(err, ShouldEqual, nil)
			So(t.Token, ShouldNotBeBlank)
			So(t.Id.Hex(), ShouldNotBeBlank)
			id := t.Id

			Convey("And user must be able to use it", func() {
				res := httptest.NewRecorder()

				// trying to get user information with scope
				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", id.Hex(), t.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)

				So(res.Code, ShouldEqual, http.StatusOK)
				a.DropDatabase()
			})
		})

		Convey("And dublicate registration should be not possible", func() {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
			req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
			req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
			a.ServeHTTP(res, req)
			So(res.Code, ShouldEqual, http.StatusBadRequest)
			a.DropDatabase()
		})

		// tokens for second user
		var tokenBody2 []byte
		var token2 gotok.Token

		Convey("Registration with other credentials should be possible", func() {
			res = httptest.NewRecorder()
			username2 := "test2@test.ru"
			res := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
			req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username2}}
			req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
			a.ServeHTTP(res, req)
			tokenBody2, _ = ioutil.ReadAll(res.Body)

			So(res.Code, ShouldEqual, http.StatusOK)
			Convey("Returned token must be valid", func() {
				err := json.Unmarshal(tokenBody2, &token2)
				So(err, ShouldEqual, nil)
				So(token2.Token, ShouldNotBeBlank)
				So(token2.Id.Hex(), ShouldNotBeBlank)
				Convey("And user must be able to use it", func() {
					res = httptest.NewRecorder()

					id := token2.Id
					reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", id.Hex(), token2.Token)
					req, _ := http.NewRequest("GET", reqUrl, nil)
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusOK)
					a.DropDatabase()
				})
				Convey("User should be able to send message", func() {

					messageText2 := "keeeeeeeee"
					messageText3 := "sheeee"
					messageText4 := "jsdkdsfklsdk"
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					// we are sending message from user2 to user1
					reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token1.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("PUT", reqUrl, nil)
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					req.PostForm = url.Values{}
					req.PostForm.Add(FORM_TEXT, messageText)
					a.ServeHTTP(res, req)
					So(res.Code, ShouldEqual, http.StatusOK)

					time.Sleep(10 * time.Millisecond)

					reqUrl = fmt.Sprintf("/api/user/%s/messages/?token=%s", token1.Id.Hex(), token2.Token)
					req, _ = http.NewRequest("PUT", reqUrl, nil)
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					req.PostForm = url.Values{}
					req.PostForm.Add(FORM_TEXT, messageText2)
					a.ServeHTTP(res, req)
					So(res.Code, ShouldEqual, http.StatusOK)
					time.Sleep(10 * time.Millisecond)

					reqUrl = fmt.Sprintf("/api/user/%s/messages/?token=%s", token1.Id.Hex(), token2.Token)
					req, _ = http.NewRequest("PUT", reqUrl, nil)
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					req.PostForm = url.Values{}
					req.PostForm.Add(FORM_TEXT, messageText3)
					a.ServeHTTP(res, req)
					So(res.Code, ShouldEqual, http.StatusOK)

					time.Sleep(10 * time.Millisecond)
					reqUrl = fmt.Sprintf("/api/user/%s/messages/?token=%s", token1.Id.Hex(), token2.Token)
					req, _ = http.NewRequest("PUT", reqUrl, nil)
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					req.PostForm = url.Values{}
					req.PostForm.Add(FORM_TEXT, messageText4)
					a.ServeHTTP(res, req)
					So(res.Code, ShouldEqual, http.StatusOK)

					var foundMessage Message
					Convey("And that message should be in messages", func() {
						res = httptest.NewRecorder()

						// we are requesting messages for user1 from user2
						reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token2.Id.Hex(), token1.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						time.Sleep(time.Millisecond * 20) // waiting for async message send
						a.ServeHTTP(res, req)
						messagesBody, _ := ioutil.ReadAll(res.Body)
						m := []Message{}
						So(res.Code, ShouldEqual, http.StatusOK)
						err := json.Unmarshal(messagesBody, &m)
						So(err, ShouldEqual, nil)
						for _, value := range m {
							foundMessage = value
							break
						}
						So(foundMessage.Destination, ShouldEqual, token1.Id)
						So(foundMessage.Origin, ShouldEqual, token2.Id)
						So(foundMessage.Text, ShouldEqual, messageText)

						So(m[0].Text, ShouldEqual, messageText)
						So(m[1].Text, ShouldEqual, messageText2)
						So(m[2].Text, ShouldEqual, messageText3)
						So(m[3].Text, ShouldEqual, messageText4)

						Convey("So user could remove it", func() {
							res = httptest.NewRecorder()
							reqUrl := fmt.Sprintf("/api/message/%s/?token=%s", foundMessage.Id.Hex(), token1.Token)
							req, _ := http.NewRequest("DELETE", reqUrl, nil)
							a.ServeHTTP(res, req)
							So(res.Code, ShouldEqual, http.StatusOK)
							Convey("And it should not be in messages now", func() {
								res = httptest.NewRecorder()

								// we are requesting messages for user1 from user2
								reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token2.Id.Hex(), token1.Token)
								req, _ := http.NewRequest("GET", reqUrl, nil)
								a.ServeHTTP(res, req)
								a.DropDatabase()
								m := []Message{}
								So(res.Code, ShouldEqual, http.StatusOK)
								decoder := json.NewDecoder(res.Body)
								err := decoder.Decode(&m)
								So(err, ShouldBeNil)
								So(len(m), ShouldEqual, 3)
							})
						})
					})
				})
				Convey("User should be able to add guests", func() {
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					reqUrl := fmt.Sprintf("/api/user/%s/guests/?token=%s", token2.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("PUT", reqUrl, nil)
					req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusOK)
					Convey("Other user should now be in guests", func() {
						res = httptest.NewRecorder()

						reqUrl := fmt.Sprintf("/api/user/%s/guests/?token=%s", token1.Id.Hex(), token1.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						a.ServeHTTP(res, req)
						a.DropDatabase()

						So(res.Code, ShouldEqual, http.StatusOK)
						u := []User{}
						userBody, _ := ioutil.ReadAll(res.Body)
						err := json.Unmarshal(userBody, &u)

						So(err, ShouldEqual, nil)
						found := false
						for _, value := range u {
							if value.Id == token2.Id {
								found = true
							}
						}
						So(found, ShouldBeTrue)
					})

					Convey("Second addition", func() {
						res = httptest.NewRecorder()

						json.Unmarshal(tokenBody, &token1)

						reqUrl := fmt.Sprintf("/api/user/%s/guests/?token=%s", token2.Id.Hex(), token2.Token)
						req, _ := http.NewRequest("PUT", reqUrl, nil)
						req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
						req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
						a.ServeHTTP(res, req)
						a.DropDatabase()

						So(res.Code, ShouldEqual, http.StatusOK)
					})
				})
				Convey("User should be able to add to blacklist", func() {
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					reqUrl := fmt.Sprintf("/api/user/%s/blacklist/?token=%s", token2.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("POST", reqUrl, nil)
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusOK)
					Convey("Other user should now be in blacklist", func() {
						res = httptest.NewRecorder()

						reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token2.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						a.ServeHTTP(res, req)

						So(res.Code, ShouldEqual, http.StatusOK)
						u := User{}
						userBody, _ := ioutil.ReadAll(res.Body)
						err := json.Unmarshal(userBody, &u)

						So(err, ShouldEqual, nil)
						So(u.Blacklist, ShouldContain, token1.Id)
						Convey("User should be able to send message", func() {
							res = httptest.NewRecorder()

							json.Unmarshal(tokenBody, &token1)

							// we are sending message from user2 to user1
							reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token1.Id.Hex(), token2.Token)
							req, _ := http.NewRequest("PUT", reqUrl, nil)
							req.PostForm = url.Values{FORM_TEXT: {messageText}}
							req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
							a.ServeHTTP(res, req)
							So(res.Code, ShouldEqual, http.StatusOK)
							var foundMessage Message
							Convey("So unread messages should equal 1", func() {
								time.Sleep(time.Millisecond * 10)
								res = httptest.NewRecorder()
								reqUrl = fmt.Sprintf("/api/user/%s/unread/?token=%s", token1.Id.Hex(), token1.Token)
								req, _ = http.NewRequest("GET", reqUrl, nil)
								a.ServeHTTP(res, req)
								a.DropDatabase()
								c := &UnreadCount{}
								So(res.Code, ShouldEqual, http.StatusOK)
								decoder := json.NewDecoder(res.Body)
								err := decoder.Decode(c)
								So(err, ShouldBeNil)
								So(c.Count, ShouldEqual, 1)
							})
							Convey("And that message should be in messages", func() {
								res = httptest.NewRecorder()
								// we are requesting messages for user1 from user2
								reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token2.Id.Hex(), token1.Token)
								req, _ := http.NewRequest("GET", reqUrl, nil)
								time.Sleep(time.Millisecond * 5) // waiting for async message send
								a.ServeHTTP(res, req)
								messagesBody, _ := ioutil.ReadAll(res.Body)
								m := []Message{}
								So(res.Code, ShouldEqual, http.StatusOK)
								err := json.Unmarshal(messagesBody, &m)
								So(err, ShouldEqual, nil)
								for _, value := range m {
									if value.Text == messageText {
										foundMessage = value
									}
								}
								So(foundMessage.Destination, ShouldEqual, token1.Id)
								So(foundMessage.Origin, ShouldEqual, token2.Id)
								So(foundMessage.Text, ShouldEqual, messageText)

								Convey("So user could mark it as read", func() {
									reqUrl := fmt.Sprintf("/api/message/%s/read?token=%s", foundMessage.Id.Hex(), token1.Token)
									req, _ := http.NewRequest("POST", reqUrl, nil)
									a.ServeHTTP(res, req)
									So(res.Code, ShouldEqual, http.StatusOK)
									Convey("So unread messages should equal zero", func() {
										res = httptest.NewRecorder()
										reqUrl = fmt.Sprintf("/api/user/%s/unread/?token=%s", token1.Id.Hex(), token1.Token)
										req, _ = http.NewRequest("GET", reqUrl, nil)
										a.ServeHTTP(res, req)
										a.DropDatabase()
										c := &UnreadCount{}
										So(res.Code, ShouldEqual, http.StatusOK)
										decoder := json.NewDecoder(res.Body)
										err := decoder.Decode(c)
										So(err, ShouldBeNil)
										So(c.Count, ShouldEqual, 0)
									})
								})

								Convey("So user could remove it", func() {
									res = httptest.NewRecorder()
									reqUrl := fmt.Sprintf("/api/message/%s/?token=%s", foundMessage.Id.Hex(), token1.Token)
									req, _ := http.NewRequest("DELETE", reqUrl, nil)
									a.ServeHTTP(res, req)
									So(res.Code, ShouldEqual, http.StatusOK)
									res = httptest.NewRecorder()

									// we are requesting messages for user1 from user2
									reqUrl = fmt.Sprintf("/api/user/%s/messages/?token=%s", token2.Id.Hex(), token1.Token)
									req, _ = http.NewRequest("GET", reqUrl, nil)
									a.ServeHTTP(res, req)
									a.DropDatabase()
									m := []Message{}
									So(res.Code, ShouldEqual, http.StatusOK)
									decoder := json.NewDecoder(res.Body)
									err := decoder.Decode(&m)
									So(err, ShouldBeNil)
									So(len(m), ShouldEqual, 0)
								})
							})
						})
						Convey("Other user should be able to send message", func() {
							res = httptest.NewRecorder()

							json.Unmarshal(tokenBody, &token1)

							// we are sending message from user1 to user2
							reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token2.Id.Hex(), token1.Token)
							req, _ := http.NewRequest("PUT", reqUrl, nil)
							req.PostForm = url.Values{FORM_TEXT: {messageText}}
							req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
							a.ServeHTTP(res, req)
							So(res.Code, ShouldEqual, http.StatusOK)

							Convey("But it should not be in inbox", func() {
								time.Sleep(5 * time.Millisecond)
								res = httptest.NewRecorder()

								// we are requesting messages for user2 from user1
								reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token2.Id.Hex(), token1.Token)
								req, _ := http.NewRequest("GET", reqUrl, nil)
								time.Sleep(time.Millisecond * 5) // waiting for async message send
								a.ServeHTTP(res, req)
								a.DropDatabase()
								m := []Message{}
								So(res.Code, ShouldEqual, http.StatusOK)
								decoder := json.NewDecoder(res.Body)
								err := decoder.Decode(&m)
								So(err, ShouldBeNil)
								So(len(m), ShouldEqual, 0)
							})
						})
						Convey("Other user now should not be able to get information", func() {
							res = httptest.NewRecorder()

							reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token1.Token)
							req, _ := http.NewRequest("GET", reqUrl, nil)
							a.ServeHTTP(res, req)

							a.DropDatabase()
							So(res.Code, ShouldEqual, http.StatusMethodNotAllowed)
						})
						Convey("Then user should be able to remove other user from blacklist", func() {
							reqUrl := fmt.Sprintf("/api/user/%s/blacklist/?token=%s", token2.Id.Hex(), token2.Token)
							req, _ := http.NewRequest("DELETE", reqUrl, nil)
							req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
							req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
							a.ServeHTTP(res, req)
							So(res.Code, ShouldEqual, http.StatusOK)
							Convey("Other user now should not be in blacklist", func() {
								res = httptest.NewRecorder()

								reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token2.Token)
								req, _ := http.NewRequest("GET", reqUrl, nil)
								a.ServeHTTP(res, req)

								So(res.Code, ShouldEqual, http.StatusOK)
								u := User{}
								userBody, _ := ioutil.ReadAll(res.Body)
								err := json.Unmarshal(userBody, &u)

								So(err, ShouldEqual, nil)
								So(u.Blacklist, ShouldNotContain, token1.Id)
								Convey("Other user now should be able to get information", func() {
									res = httptest.NewRecorder()

									reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token1.Token)
									req, _ := http.NewRequest("GET", reqUrl, nil)
									a.ServeHTTP(res, req)

									a.DropDatabase()
									So(res.Code, ShouldEqual, http.StatusOK)
								})
							})
						})
					})
				})
				Convey("And user should be able to add other user to own favorites", func() {
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					reqUrl := fmt.Sprintf("/api/user/%s/fav/?token=%s", token2.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("POST", reqUrl, nil)
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusOK)
					Convey("Other user should not see users favorites", func() {
						res = httptest.NewRecorder()

						reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token1.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						a.ServeHTTP(res, req)
						a.DropDatabase()

						So(res.Code, ShouldEqual, http.StatusOK)
						u := User{}
						userBody, _ := ioutil.ReadAll(res.Body)
						err := json.Unmarshal(userBody, &u)

						So(err, ShouldEqual, nil)
						So(u.Favorites, ShouldNotContain, token1.Id)
					})
					Convey("Other user should now be in favorites", func() {
						res = httptest.NewRecorder()

						reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token2.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						a.ServeHTTP(res, req)
						a.DropDatabase()

						So(res.Code, ShouldEqual, http.StatusOK)
						u := User{}
						userBody, _ := ioutil.ReadAll(res.Body)
						err := json.Unmarshal(userBody, &u)

						So(err, ShouldEqual, nil)
						So(u.Favorites, ShouldContain, token1.Id)
					})
					Convey("Other user should be in full favorites list", func() {
						res = httptest.NewRecorder()

						reqUrl := fmt.Sprintf("/api/user/%s/fav/?token=%s", token2.Id.Hex(), token2.Token)
						req, _ := http.NewRequest("GET", reqUrl, nil)
						a.ServeHTTP(res, req)
						a.DropDatabase()

						So(res.Code, ShouldEqual, http.StatusOK)
						u := []User{}
						userBody, _ := ioutil.ReadAll(res.Body)
						err := json.Unmarshal(userBody, &u)

						So(err, ShouldEqual, nil)
						found := false
						for _, value := range u {
							if value.Id == token1.Id {
								found = true
							}
						}
						So(found, ShouldBeTrue)
					})
					Convey("Then user should be able to remove other user from favorites", func() {
						reqUrl := fmt.Sprintf("/api/user/%s/fav/?token=%s", token2.Id.Hex(), token2.Token)
						req, _ := http.NewRequest("DELETE", reqUrl, nil)
						req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
						req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
						a.ServeHTTP(res, req)
						So(res.Code, ShouldEqual, http.StatusOK)
						Convey("Other user now should not be in favorites", func() {
							res = httptest.NewRecorder()

							reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token2.Id.Hex(), token2.Token)
							req, _ := http.NewRequest("GET", reqUrl, nil)
							a.ServeHTTP(res, req)
							a.DropDatabase()

							So(res.Code, ShouldEqual, http.StatusOK)
							u := User{}
							userBody, _ := ioutil.ReadAll(res.Body)
							err := json.Unmarshal(userBody, &u)

							So(err, ShouldEqual, nil)
							So(u.Favorites, ShouldNotContain, token1.Id)
						})
					})
				})
				Convey("And user should not be able to modify other user favorites", func() {
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)
					id1 := token1.Id
					id2 := token2.Id

					reqUrl := fmt.Sprintf("/api/user/%s/fav/?token=%s", id1.Hex(), token2.Token)
					req, _ := http.NewRequest("POST", reqUrl, nil)
					req.PostForm = url.Values{FORM_TARGET: {id2.Hex()}}
					req.Header.Add(ContentTypeHeader, "x-www-form-urlencoded")
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusMethodNotAllowed)
					a.DropDatabase()
				})
			})
		})
	})
}
