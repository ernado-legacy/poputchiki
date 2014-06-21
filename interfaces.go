package main

import (
	"github.com/ernado/gotok"
	"github.com/ernado/poputchiki-api/weed"
	"labix.org/v2/mgo/bson"
	"net/http"
)

type WebpAccept bool

type VideoAccept string

type AudioAccept string

var (
	VA_WEBM VideoAccept = "webm"
	VA_MP4  VideoAccept = "mp4"
	AA_ACC  AudioAccept = "acc"
	AA_OGG  AudioAccept = "ogg"
)

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

	AddPhoto(user bson.ObjectId, imageJpeg File, imageWebp File, thumbnailJpeg File, thumbnailWebp File, desctiption string) (*Photo, error)
	// api/photo/:id
	GetPhoto(photo bson.ObjectId) (*Photo, error)
	// api/photo/:id/comment
	AddCommentToPhoto(user bson.ObjectId, photo bson.ObjectId, c *Comment) error

	AddAlbum(user bson.ObjectId, album *Album) (*Album, error)
	AddPhotoToAlbum(user bson.ObjectId, album bson.ObjectId, photo bson.ObjectId) error

	AddFile(file *File) (*File, error)
	AddVideo(video *Video) (*Video, error)
	GetVideo(id bson.ObjectId) *Video
	AddAudio(audio *Audio) (*Audio, error)
	GetAudio(id bson.ObjectId) *Audio

	AddStripeItem(user bson.ObjectId, media interface{}) (*StripeItem, error)
	GetStripeItem(id bson.ObjectId) (*StripeItem, error)
	GetStripe(count, offset int) ([]*StripeItem, error)

	Search(q *SearchQuery, count, offset int) ([]*User, error)
	SearchStatuses(q *SearchQuery, count, offset int) ([]*StatusUpdate, error)

	NewConfirmationToken(id bson.ObjectId) *EmailConfirmationToken
	GetConfirmationToken(token string) *EmailConfirmationToken
}

type RealtimeInterface interface {
	Push(id bson.ObjectId, event interface{}) error
	RealtimeHandler(w http.ResponseWriter, r *http.Request, t *gotok.Token) (int, []byte)
}

type PrepareInterface interface {
	Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, audio AudioAccept) error
}

// mgClient := mailgun.New(mailKey)
// message := ConfirmationMail{}
// message.Destination = u.Email
// message.Mail = "http://poputchiki.ru/api/confirm/email/" + confTok.Token
// _, err = mgClient.Send(message)
type MailSender interface {
	Send()
}
