package models

import (
	"github.com/ernado/gotok"
	"github.com/ernado/weed"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
)

// webp accept flag
type WebpAccept bool

// video format accept
type VideoAccept string

// audio format accept
type AudioAccept string

var (
	VaWebm VideoAccept = "webm"
	VaMp4  VideoAccept = "mp4"
	AaAac  AudioAccept = "acc"
	AaOgg  AudioAccept = "ogg"
)

type SearchResult struct {
	Result interface{} `json:"result"`
	Count  int         `json:"count"`
}

type DataBase interface {
	// GetAll() []*User
	GetUsername(username string) *User
	Get(id bson.ObjectId) *User
	Add(u *User) error
	Update(id bson.ObjectId, update bson.M) (*User, error)
	// Delete(id bson.ObjectId) error
	AddToFavorites(id bson.ObjectId, favId bson.ObjectId) error
	RemoveFromFavorites(id bson.ObjectId, favId bson.ObjectId) error
	GetFavorites(id bson.ObjectId) []*User

	AddGuest(id bson.ObjectId, guest bson.ObjectId) error
	GetAllGuests(id bson.ObjectId) ([]*User, error)
	GetAllGuestUsers(id bson.ObjectId) ([]*GuestUser, error)

	AddMessage(m *Message) error
	GetMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) ([]*Message, error)
	GetMessage(id bson.ObjectId) (message *Message, err error)
	RemoveMessage(id bson.ObjectId) error
	GetChats(id bson.ObjectId) ([]*User, error)
	SetRead(user, id bson.ObjectId) error
	SetReadMessagesFromUser(userReciever bson.ObjectId, userOrigin bson.ObjectId) error
	GetUnreadCount(id bson.ObjectId) (int, error)

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
	AddLikeStatus(user bson.ObjectId, target bson.ObjectId) error
	RemoveLikeStatus(user bson.ObjectId, target bson.ObjectId) error
	GetLikesStatus(id bson.ObjectId) []*User
	// api/status/:id/comment/:id

	AddPhoto(user bson.ObjectId, imageJpeg File, imageWebp File, thumbnailJpeg File, thumbnailWebp File, desctiption string) (*Photo, error)

	// api/photo/:id
	GetPhoto(photo bson.ObjectId) (*Photo, error)
	RemovePhoto(user bson.ObjectId, id bson.ObjectId) error
	SearchPhoto(q *SearchQuery, count, offset int) ([]*Photo, error)
	AddLikePhoto(user bson.ObjectId, target bson.ObjectId) error
	RemoveLikePhoto(user bson.ObjectId, target bson.ObjectId) error
	GetLikesPhoto(id bson.ObjectId) []*User
	GetUserPhoto(user bson.ObjectId) ([]*Photo, error)

	AddFile(file *File) (*File, error)

	AddVideo(video *Video) (*Video, error)
	GetVideo(id bson.ObjectId) *Video
	GetUserVideo(id bson.ObjectId) *Video
	AddLikeVideo(user bson.ObjectId, target bson.ObjectId) error
	RemoveLikeVideo(user bson.ObjectId, target bson.ObjectId) error
	GetLikesVideo(id bson.ObjectId) []*User
	UpdateVideoWebm(id bson.ObjectId, fid string) error
	UpdateVideoMpeg(id bson.ObjectId, fid string) error

	AddAudio(audio *Audio) (*Audio, error)
	GetAudio(id bson.ObjectId) *Audio
	UpdateAudioAAC(id bson.ObjectId, fid string) error
	UpdateAudioOGG(id bson.ObjectId, fid string) error

	AddStripeItem(i *StripeItem, media interface{}) (*StripeItem, error)
	GetStripeItem(id bson.ObjectId) (*StripeItem, error)
	GetStripe(count, offset int) ([]*StripeItem, error)

	Search(q *SearchQuery, count, offset int) ([]*User, int, error)
	SearchStatuses(q *SearchQuery, count, offset int) ([]*StatusUpdate, error)

	NewConfirmationToken(id bson.ObjectId) *EmailConfirmationToken
	GetConfirmationToken(token string) *EmailConfirmationToken
	NewConfirmationTokenValue(id bson.ObjectId, token string) *EmailConfirmationToken
	ConfirmEmail(id bson.ObjectId) error
	ConfirmPhone(id bson.ObjectId) error

	UpdateAllStatuses() (*mgo.ChangeInfo, error)
	SetLastActionNow(id bson.ObjectId) error

	GetCities(start, country string) (cities []string, err error)
	GetCountries(start string) (countries []string, err error)
	GetPlaces(start string) (places []string, err error)

	Init()
	Drop()
	Salt() string

	SetAvatar(user, avatar bson.ObjectId) error
}

type RealtimeInterface interface {
	Push(id bson.ObjectId, event interface{}) error
	RealtimeHandler(w http.ResponseWriter, r *http.Request, t *gotok.Token) (int, []byte)
}

type PrepareInterface interface {
	Prepare(adapter *weed.Adapter, webp WebpAccept, video VideoAccept, audio AudioAccept) error
}

type MailSender interface {
	Send()
}
