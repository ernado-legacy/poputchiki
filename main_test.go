package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	. "github.com/smartystreets/goconvey/convey"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
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

func TestEtcd(t *testing.T) {
	Convey("etcd connect", t, func() {
		So(loadConfig(), ShouldBeNil)
	})
}

func TestDBMethods(t *testing.T) {
	dbName = "poputchiki_dev_db"
	var err error
	a := NewApp()
	session := a.session
	u := User{}

	Convey("Database init", t, func() {
		db := NewDatabase(session)
		Reset(func() {
			a.DropDatabase()
		})
		Convey("User should be created", func() {
			u.Password = "test"
			u.Id = bson.NewObjectId()
			id := u.Id
			u.Email = "test@test.te"
			err = db.Add(&u)
			So(err, ShouldBeNil)
			Convey("Balance update", func() {
				So(db.IncBalance(id, 100), ShouldBeNil)
				So(db.DecBalance(id, 50), ShouldBeNil)
				newU := db.Get(id)
				So(newU.Balance, ShouldEqual, 50)
				So(db.DecBalance(id, 100), ShouldNotBeNil)
			})
			Convey("Status update", func() {
				db.SetOnline(id)
				newU1 := db.Get(id)
				db.SetOffline(id)
				newU2 := db.Get(id)
				So(newU1.Online, ShouldEqual, true)
				So(newU2.Online, ShouldEqual, false)
			})

			Convey("Stripe add", func() {
				video := Video{}
				video.Id = bson.NewObjectId()
				video.User = id

				s, err := db.AddStripeItem(id, video)
				So(err, ShouldBeNil)
				Convey("Stripe get", func() {
					s1, err := db.GetStripeItem(s.Id)
					So(err, ShouldBeNil)
					So(s1.Type, ShouldEqual, "video")
					dta, err := bson.Marshal(s1.Media)
					So(err, ShouldBeNil)
					So(bson.Unmarshal(dta, &video), ShouldBeNil)
				})
			})

			Convey("Status add", func() {
				text := "status hello world"
				s, err := db.AddStatus(id, text)
				So(err, ShouldBeNil)
				So(s.Text, ShouldEqual, text)
				Convey("Remove", func() {
					err = db.RemoveStatusSecure(id, s.Id)
					So(err, ShouldBeNil)
				})
				Convey("Remove is secure", func() {
					err = db.RemoveStatusSecure(bson.NewObjectId(), s.Id)
					So(err, ShouldNotBeNil)
				})
				Convey("Update", func() {
					newText := "status2"
					s1, err := db.UpdateStatusSecure(id, s.Id, newText)
					So(err, ShouldBeNil)
					s2, err := db.GetStatus(s.Id)
					So(err, ShouldBeNil)
					So(s1.Text, ShouldEqual, newText)
					So(s2.Text, ShouldEqual, newText)
					So(s1.Id, ShouldEqual, s.Id)
					So(s2.Id, ShouldEqual, s.Id)
					So(s1.Text, ShouldNotEqual, s.Text)
				})
				Convey("Exists", func() {
					s1, err := db.GetCurrentStatus(id)
					So(err, ShouldBeNil)
					So(s1.Text, ShouldEqual, text)
				})
				Convey("Actual", func() {
					newText := "status actual"
					_, err := db.AddStatus(id, newText)
					So(err, ShouldBeNil)
					s2, err := db.GetCurrentStatus(id)
					So(err, ShouldBeNil)
					So(s2.Text, ShouldEqual, newText)

					statuses, err := db.GetLastStatuses(10)
					So(err, ShouldBeNil)
					So(statuses[0].Text, ShouldEqual, newText)
				})
				Convey("Add comment", func() {
					commentText := "comment text"
					comment, err := db.AddCommentToStatus(id, s.Id, commentText)
					So(err, ShouldBeNil)
					So(comment.Text, ShouldEqual, commentText)
					Convey("Remove", func() {
						err := db.RemoveCommentFromStatusSecure(id, comment.Id)
						So(err, ShouldBeNil)
					})
					Convey("Update", func() {
						commentText2 := "batman"
						err := db.UpdateCommentToStatusSecure(id, comment.Id, commentText2)
						So(err, ShouldBeNil)
						s, err := db.GetStatus(s.Id)
						So(err, ShouldBeNil)
						correct := false
						for _, c := range s.Comments {
							if c.Id != comment.Id {
								continue
							}
							if c.Text == commentText2 {
								correct = true
							}
						}
						So(correct, ShouldBeTrue)
					})
				})

			})
			Convey("Add photo", func() {
				i := File{Id: bson.NewObjectId(), User: id}
				_, err := db.AddPhoto(id, i, i, i, i, "test")
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestUpload(t *testing.T) {
	username := "test@test.ru"
	password := "secretsecret"
	redisName = "poputchiki_test_upload"
	dbName = "poputchiki_dev_upload"
	path := "test/image.jpg"
	file, err := os.Open(path)
	a := NewApp()
	defer a.Close()
	a.DropDatabase()

	Convey("Registration with unique username and valid password should be successfull", t, func() {
		Reset(func() {
			a.DropDatabase()
		})
		So(err, ShouldBeNil)
		res := httptest.NewRecorder()
		// sending registration request
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
		a.ServeHTTP(res, req)

		// reading response
		So(res.Code, ShouldEqual, http.StatusOK)
		tokenBody, _ := ioutil.ReadAll(res.Body)
		token := &Token{}
		So(json.Unmarshal(tokenBody, token), ShouldBeNil)

		res = httptest.NewRecorder()
		album := &Album{}
		albumJs, err := json.Marshal(album)
		buf := bytes.NewReader(albumJs)
		So(err, ShouldBeNil)
		req, _ = http.NewRequest("PUT", "/api/album?token="+token.Token, buf)
		a.ServeHTTP(res, req)
		So(res.Code, ShouldEqual, http.StatusOK)
		albumJs, err = ioutil.ReadAll(res.Body)
		So(err, ShouldBeNil)
		So(json.Unmarshal(albumJs, album), ShouldBeNil)

		Convey("Request should completed", func() {
			So(err, ShouldBeNil)
			defer file.Close()
			res := httptest.NewRecorder()
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", filepath.Base(path))
			a.DropDatabase()
			So(err, ShouldBeNil)
			_, err = io.Copy(part, file)
			So(err, ShouldBeNil)
			So(writer.Close(), ShouldBeNil)
			req, err := http.NewRequest("POST", "/api/album/"+album.Id.Hex()+"/photo/?token="+token.Token, body)
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

func TestMethods(t *testing.T) {
	username := "test@test.ru"
	password := "secretsecret"
	firstname := "Ivan"
	secondname := "Pupkin"
	phone := "+79197241488"

	messageText := "hello world русский текст"

	dbName = "poputchiki_dev"
	redisName = "poputchiki_dev"

	a := NewApp()
	defer a.Close()
	a.DropDatabase()

	var tokenBody []byte
	var token1 Token

	Convey("Registration with unique username and valid password should be successfull", t, func() {
		res := httptest.NewRecorder()
		// sending registration request
		req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
		req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
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
			Convey("500 InternalServerError - database is dead", func() {
				a.Close()
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)
				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId().Hex(), token1.Token)
				req, _ := http.NewRequest("GET", reqUrl, nil)
				a.ServeHTTP(res, req)
				a = NewApp()
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusInternalServerError)
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
				a.ServeHTTP(res, req)
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusMethodNotAllowed)
			})
			Convey("500 InternalServerError - database is dead", func() {
				a.Close()
				res := httptest.NewRecorder()
				err := json.Unmarshal(tokenBody, &token1)
				So(err, ShouldEqual, nil)

				reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", bson.NewObjectId().Hex(), token1.Token)
				req, _ := http.NewRequest("PATCH", reqUrl, nil)
				req.PostForm = url.Values{FORM_FIRSTNAME: {firstname}, FORM_SECONDNAME: {secondname}, FORM_PHONE: {phone}}
				a.ServeHTTP(res, req)
				a = NewApp()
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})

		Convey("Login error handling", func() {
			Convey("404 Not found - user is nonexistent", func() {
				res := httptest.NewRecorder()
				// trying to log in
				req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
				req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {"randomemail"}}
				a.ServeHTTP(res, req)

				So(res.Code, ShouldEqual, http.StatusNotFound)
				a.DropDatabase()
			})
			Convey("404 Unauthorised - incorrect password", func() {
				res := httptest.NewRecorder()
				// trying to log in
				req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
				req.PostForm = url.Values{FORM_PASSWORD: {"randompass"}, FORM_EMAIL: {username}}
				a.ServeHTTP(res, req)

				So(res.Code, ShouldEqual, http.StatusUnauthorized)
				a.DropDatabase()
			})
			Convey("500 InternalServerError - database is dead", func() {
				a.Close()
				res := httptest.NewRecorder()
				// trying to log in
				req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
				req.PostForm = url.Values{FORM_PASSWORD: {"randompass"}, FORM_EMAIL: {"randomemail"}}
				a.ServeHTTP(res, req)
				a = NewApp()
				a.DropDatabase()
				So(res.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})

		Convey("User should be able to change information after registration", func() {
			err := json.Unmarshal(tokenBody, &token1)
			So(err, ShouldEqual, nil)

			reqUrl := fmt.Sprintf("/api/user/%s/?token=%s", token1.Id.Hex(), token1.Token)
			u := User{}
			u.Id = token1.Id
			u.FirstName = firstname
			u.SecondName = secondname
			u.Phone = phone
			uJson, err := json.Marshal(u)
			uReader := bytes.NewReader(uJson)
			So(err, ShouldBeNil)
			req, _ := http.NewRequest("PATCH", reqUrl, uReader)
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
				So(u.FirstName, ShouldEqual, firstname)
				So(u.SecondName, ShouldEqual, secondname)
				So(u.Phone, ShouldEqual, phone)
				a.DropDatabase()
			})
		})

		Convey("User should be able to log in after registration", func() {
			res := httptest.NewRecorder()
			// trying to log in
			req, _ := http.NewRequest("POST", "/api/auth/login/", nil)
			req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username}}
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
			t := Token{}
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
			a.ServeHTTP(res, req)
			So(res.Code, ShouldEqual, http.StatusBadRequest)
			a.DropDatabase()
		})

		// tokens for second user
		var tokenBody2 []byte
		var token2 Token

		Convey("Registration with other credentials should be possible", func() {
			res = httptest.NewRecorder()
			username2 := "test2@test.ru"
			res := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/auth/register/", nil)
			req.PostForm = url.Values{FORM_PASSWORD: {password}, FORM_EMAIL: {username2}}
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
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					// we are sending message from user2 to user1
					reqUrl := fmt.Sprintf("/api/user/%s/messages/?token=%s", token1.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("PUT", reqUrl, nil)
					req.PostForm = url.Values{FORM_TEXT: {messageText}}
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
							if value.Text == messageText {
								foundMessage = value
							}
						}
						So(foundMessage.Destination, ShouldEqual, token1.Id)
						So(foundMessage.Origin, ShouldEqual, token2.Id)
						So(foundMessage.Text, ShouldEqual, messageText)

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
								So(res.Code, ShouldEqual, http.StatusNotFound)
							})
						})
					})
				})
				Convey("User should be able to add guests", func() {
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					reqUrl := fmt.Sprintf("/api/user/%s/guests/?token=%s", token2.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("POST", reqUrl, nil)
					req.PostForm = url.Values{FORM_TARGET: {token1.Id.Hex()}}
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusOK)
					Convey("Other user should now be in guests", func() {
						res = httptest.NewRecorder()

						reqUrl := fmt.Sprintf("/api/user/%s/guests/?token=%s", token2.Id.Hex(), token2.Token)
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
				})
				Convey("User should be able to add to blacklist", func() {
					res = httptest.NewRecorder()

					json.Unmarshal(tokenBody, &token1)

					reqUrl := fmt.Sprintf("/api/user/%s/blacklist/?token=%s", token2.Id.Hex(), token2.Token)
					req, _ := http.NewRequest("POST", reqUrl, nil)
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
							a.ServeHTTP(res, req)
							So(res.Code, ShouldEqual, http.StatusOK)
							var foundMessage Message
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
									So(res.Code, ShouldEqual, http.StatusNotFound)
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
								So(res.Code, ShouldEqual, http.StatusNotFound)
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
					a.ServeHTTP(res, req)

					So(res.Code, ShouldEqual, http.StatusMethodNotAllowed)
					a.DropDatabase()
				})
			})
		})
	})
}
