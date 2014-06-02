package main

import (
	"labix.org/v2/mgo/bson"
	"net/http"
)

type WebpAccept bool

type UserDB interface {
	// GetAll() []*User
	GetUsername(username string) *User
	Get(id bson.ObjectId) *User
	Add(u *User) error
	Update(u *User) error
	// Delete(id bson.ObjectId) error
	AddToFavorites(id bson.ObjectId, favId bson.ObjectId) error
	RemoveFromFavorites(id bson.ObjectId, favId bson.ObjectId) error
	GetFavorites(id bson.ObjectId) []*User

	AddGuest(id bson.ObjectId, guest bson.ObjectId) error
	GetAllGuests(id bson.ObjectId) ([]*User, error)

	AddMessage(m *Message) error
	GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) ([]*Message, error)
	GetMessage(id bson.ObjectId) (message *Message, err error)
	RemoveMessage(id bson.ObjectId) error

	AddToBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error
	RemoveFromBlacklist(id bson.ObjectId, blacklisted bson.ObjectId) error

	IncBalance(id bson.ObjectId, amount uint) error
	DecBalance(id bson.ObjectId, amount uint) error

	SetOnline(id bson.ObjectId) error
	SetOffline(id bson.ObjectId) error

	// statuses
	// api/user/:id/status/
	GetCurrentStatus(user bson.ObjectId) (status *StatusUpdate, err error)
	// api/status/:id/
	GetStatus(id bson.ObjectId) (status *StatusUpdate, err error)
	GetLastStatuses(count int) (status []*StatusUpdate, err error)
	AddStatus(u bson.ObjectId, text string) (*StatusUpdate, error)
	UpdateStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) (*StatusUpdate, error)
	RemoveStatusSecure(user bson.ObjectId, id bson.ObjectId) error
	// api/status/:id/comment/:id
	AddCommentToStatus(user bson.ObjectId, status bson.ObjectId, text string) (*Comment, error)
	RemoveCommentFromStatusSecure(user bson.ObjectId, id bson.ObjectId) error
	UpdateCommentToStatusSecure(user bson.ObjectId, id bson.ObjectId, text string) error

	AddPhoto(user bson.ObjectId, imageJpeg File, imageWebp File, desctiption string) (*Photo, error)
	// api/photo/:id
	GetPhoto(photo bson.ObjectId) (*Photo, error)
	// api/photo/:id/comment
	AddCommentToPhoto(user bson.ObjectId, photo bson.ObjectId, c *Comment) error

	AddAlbum(user bson.ObjectId, album *Album) (*Album, error)
	AddPhotoToAlbum(user bson.ObjectId, album bson.ObjectId, photo bson.ObjectId) error
}

type TokenStorage interface {
	Get(hexToken string) (*Token, error)
	Generate(user *User) (*Token, error)
	Remove(token *Token) error
}

type RealtimeInterface interface {
	Push(id bson.ObjectId, event interface{}) error
	RealtimeHandler(w http.ResponseWriter, r *http.Request, t *Token) (int, []byte)
}
