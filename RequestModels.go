package main

import (
	"reflect"
	"strings"

	"github.com/JojiiOfficial/WhShareServer/models"
	log "github.com/sirupsen/logrus"
)

//TODO move to models package

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
	Sources []models.Source `json:"sources,omitempty"`
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
