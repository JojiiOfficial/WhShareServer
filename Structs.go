package main

import (
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ------------- Database structs ----------------

//Source a webhook source
type Source struct {
	PkID         uint32 `db:"pk_id" orm:"pk,ai" json:"-"`
	Name         string `db:"name" json:"name"`
	SourceID     string `db:"sourceID" json:"sourceID"`
	Description  string `db:"description" json:"description"`
	Secret       string `db:"secret" json:"secret"`
	CreatorID    uint32 `db:"creator" json:"-"`
	CreationTime string `db:"creationTime" json:"crTime"`
	IsPrivate    bool   `db:"private" json:"isPrivate"`
	Mode         uint8  `db:"mode" json:"mode"`
	Creator      User   `db:"-" json:"-"`
}

//LoginSession a login session
type LoginSession struct {
	PkID    uint32 `db:"pk_id" orm:"pk,ai"`
	UserID  uint32 `db:"userID"`
	Token   string `db:"sessionToken"`
	Created string `db:"created"`
	IsValid bool   `db:"isValid"`
	User    User
}

//Webhook the actual webhook from a server
type Webhook struct {
	PkID     uint32 `db:"pk_id" orm:"pk,ai"`
	SourceID uint32 `db:"sourceID"`
	Headers  string `db:"header"`
	Payload  string `db:"payload"`
	Received string `db:"received"`
}

//Role the role of a user
type Role struct {
	PkID             uint32 `db:"pk_id" orm:"pk,ai"`
	Name             string `db:"name"`
	MaxPrivSources   int    `db:"maxPrivSources"`
	MaxPubSources    int    `db:"maxPubSources"`
	MaxSubscriptions int    `db:"maxSubscriptions"`
	MaxHookCalls     int    `db:"maxHookCalls"`
	MaxTraffic       int    `db:"maxTraffic"`
}

// ------------- REST structs ----------------

//-----> Requests

type credentialRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

type sourceAddRequest struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"descr"`
	Private     bool   `json:"private"`
	Mode        uint8  `json:"mode"`
}

type subscriptionRequest struct {
	Token       string `json:"token"`
	SourceID    string `json:"sid"`
	CallbackURL string `json:"cbUrl"`
}

type unsubscribeRequest struct {
	SubscriptionID string `json:"sid"`
}

type subscriptionUpdateCallbackRequest struct {
	Token          string `json:"token"`
	SubscriptionID string `json:"subID"`
	CallbackURL    string `json:"cbUrl"`
}

type sourceRequest struct {
	Token    string `json:"token"`
	SourceID string `json:"sid,omitempty"`
	Content  string `json:"content,omitempty"`
}

type tokenOnlyRequest struct {
	Token string `json:"token"`
}

//-----> Responses

type loginResponse struct {
	Token string `json:"token"`
}

type sourceAddResponse struct {
	Secret   string `json:"secret"`
	SourceID string `json:"id"`
}

type subscriptionResponse struct {
	Message        string `json:"message,omitempty"`
	SubscriptionID string `json:"sid"`
	Name           string `json:"name"`
	Mode           uint8  `json:"mode"`
}

type listSourcesResponse struct {
	Sources []Source `json:"sources,omitempty"`
}

//Functions
func isStructInvalid(x interface{}) bool {
	s := reflect.TypeOf(x)
	for i := s.NumField() - 1; i >= 0; i-- {
		e := reflect.ValueOf(x).Field(i)

		if hasEmptyValue(e) {
			return true
		}
	}
	return false
}

func hasEmptyValue(e reflect.Value) bool {
	switch e.Type().Kind() {
	case reflect.String:
		if e.String() == "" || strings.Trim(e.String(), " ") == "" {
			return true
		}
	case reflect.Array:
		for j := e.Len() - 1; j >= 0; j-- {
			isEmpty := hasEmptyValue(e.Index(j))
			if isEmpty {
				return true
			}
		}
	case reflect.Slice:
		return isStructInvalid(e)

	case
		reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer,
		reflect.Uint64, reflect.Uint, reflect.Uint8, reflect.Bool,
		reflect.Struct, reflect.Int64, reflect.Int:
		{
			return false
		}
	default:
		log.Error(e.Type().Kind(), e)
		return true
	}
	return false
}
