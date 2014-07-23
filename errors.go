package main

import (
	"net/http"
)

type Error struct {
	Code int    `json:"status"`
	Text string `json:"text"`
}

var (
	ErrorNotAllowed            = Error{http.StatusMethodNotAllowed, "You are trying to modify others user information"}
	ErrorBlacklisted           = Error{http.StatusMethodNotAllowed, "You are blacklisted"}
	ErrorAuth                  = Error{http.StatusUnauthorized, "Not authenticated"}
	ErrorMarshal               = Error{http.StatusInternalServerError, "Unable to unmarshal data"}
	ErrorUserNotFound          = Error{http.StatusNotFound, "User not found"}
	ErrorObjectNotFound        = Error{http.StatusNotFound, "Object not found"}
	ErrorBadId                 = Error{http.StatusBadRequest, "Bad user id"}
	ErrorBadRequest            = Error{http.StatusBadRequest, "Bad request"}
	ErrorInsufficentFunds      = Error{http.StatusPaymentRequired, "Insufficent funds"}
	ErrorBackend               = Error{http.StatusInternalServerError, "Internal server error"}
	ErrorUserAlreadyRegistered = Error{http.StatusBadRequest, "User already registered"}
)
