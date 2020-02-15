package main

import (
	"fmt"
	"reflect"
	"strings"
)

// ------------- Database structs ----------------

//User user in db
type User struct {
	Pkid      uint32 `db:"pk_id" orm:"pk,ai"`
	Username  string `db:"username"`
	CreatedAt string `db:"createdAt"`
	IsValid   bool   `db:"isValid"`
}

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

//Subscription the subscription a user made
type Subscription struct {
	PkID           uint32 `db:"pk_id" orm:"pk,ai"`
	SubscriptionID string `db:"subscriptionID"`
	UserID         uint32 `db:"subscriber"`
	Source         uint32 `db:"source"`
	CallbackURL    string `db:"callbackURL"`
	Time           string `db:"time"`
	LastTrigger    string `db:"lastTrigger"`
}

// ------------- REST structs ----------------

//-----> Requests

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"pass"`
}

type sourceAddRequest struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"descr"`
	Private     bool   `json:"private"`
}

type subscriptionRequest struct {
	Token       string `json:"token"`
	SourceID    string `json:"sid"`
	CallbackURL string `json:"cburl"`
}

type unsubscribeRequest struct {
	SubscriptionID string `json:"sid"`
}

type listSourcesRequest struct {
	Token    string `json:"token"`
	SourceID string `json:"sid,omitempty"`
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
}

type listSourcesResponse struct {
	Sources []Source `json:"sources,omitempty"`
}

//Functions
func isStructInvalid(x interface{}) bool {
	s := reflect.TypeOf(x)
	for i := s.NumField() - 1; i >= 0; i-- {
		e := reflect.ValueOf(x).Field(i)

		if isEmptyValue(e) {
			return true
		}
	}
	return false
}

func isEmptyValue(e reflect.Value) bool {
	switch e.Type().Kind() {
	case reflect.String:
		if e.String() == "" || strings.Trim(e.String(), " ") == "" {
			return true
		}
	case reflect.Array:
		for j := e.Len() - 1; j >= 0; j-- {
			isEmpty := isEmptyValue(e.Index(j))
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
		fmt.Println(e.Type().Kind(), e)
		return true
	}
	return false
}
