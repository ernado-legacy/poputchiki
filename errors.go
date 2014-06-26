package main

import (
	"net/http"
)

type Error struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

var (
	ErrorNotAllowed       = Error{http.StatusMethodNotAllowed, "Not allowed"}
	ErrorBlacklisted      = Error{http.StatusMethodNotAllowed, "You are blacklisted"}
	ErrorAuth             = Error{http.StatusUnauthorized, "Not authenticated"}
	ErrorMarshal          = Error{http.StatusInternalServerError, "Unable to unmarshal data"}
	ErrorUserNotFound     = Error{http.StatusNotFound, "User not found"}
	ErrorBadId            = Error{http.StatusBadRequest, "Bad user id"}
	ErrorBadRequest       = Error{http.StatusBadRequest, "Bad request"}
	ErrorInsufficentFunds = Error{http.StatusPaymentRequired, "Insufficent funds"}
	ErrorBackend          = Error{http.StatusInternalServerError, "Internal server error"}
)
