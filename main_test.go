package main

import (
	"encoding/json"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

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
			req, _ := http.NewRequest("PATCH", reqUrl, nil)
			req.PostForm = url.Values{FORM_FIRSTNAME: {firstname}, FORM_SECONDNAME: {secondname}, FORM_PHONE: {phone}}
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
