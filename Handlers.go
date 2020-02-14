package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

var (
	db *dbhelper.DBhelper
)

func unsubscribe(w http.ResponseWriter, r *http.Request) {
	var request unsubscribeRequest
	if !handleUserInput(w, r, &request) {
		return
	}
	if len(request.SubscriptionID) != 32 {
		sendError("input missing wrong lengh", w, WrongInputFormatError, 422)
		return
	}
	err := removeSubscription(db, request.SubscriptionID)
	if err != nil {
		sendError("sever error", w, ServerError, 500)
		return
	}
	handleError(sendSuccess(w, ResponseSuccess), w, ServerError, 500)
	return
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	var request subscriptionRequest
	if !handleUserInput(w, r, &request) {
		return
	}
	token := request.Token
	if token == "-" {
		token = ""
	}

	if isStructInvalid(request) || (len(request.Token) > 0 && len(request.Token) != 64) {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	//Determine the user
	userID := uint32(1)
	var user *User
	var err error
	if len(token) > 0 {
		user, err = getUserIDFromSession(db, token)
		if err != nil {
			fmt.Println(err.Error())
			userID = 1
		} else {
			userID = user.Pkid
		}
	}

	//The source to get subbed
	source, err := getSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, WrongInputFormatError, 422)
		fmt.Println(err.Error())
		return
	}

	var response subscriptionResponse

	if userID > 1 {
		is, err := user.isSubscribedTo(db, source.PkID)
		if err != nil {
			sendError("input missing", w, ServerError, 500)
			fmt.Println(err.Error())
			return
		}
		if is {
			response = subscriptionResponse{
				Message: "You can only subscribe one time to a source",
				Status:  "error",
			}
			handleError(sendSuccess(w, response), w, ServerError, 500)
			return
		}
	}

	if source.IsPrivate && source.CreatorID == userID || !source.IsPrivate {
		sub := Subscription{
			Source:      source.PkID,
			CallbackURL: request.CallbackURL,
			UserID:      userID,
		}
		err := sub.insert(db)
		if err != nil {
			fmt.Println(err.Error())
			sendError("internal error", w, ServerError, 500)
			return
		}
		response = subscriptionResponse{
			Status:         ResponseSuccessStr,
			SubscriptionID: sub.SubscriptionID,
			Name:           source.Name,
		}
	} else {
		response = subscriptionResponse{
			Status:  ResponseErrorStr,
			Message: "Not allowed",
		}
	}
	handleError(sendSuccess(w, response), w, ServerError, 500)
}

//-> /source/add
func createSource(w http.ResponseWriter, r *http.Request) {
	var request sourceAddRequest
	if !handleUserInput(w, r, &request) {
		return
	}
	if isStructInvalid(request) || len(request.Token) != 64 {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	user, err := getUserIDFromSession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}

	source := &Source{
		Creator:     *user,
		IsPrivate:   request.Private,
		Name:        request.Name,
		Description: request.Description,
	}

	err = source.insert(db)
	if err != nil {
		fmt.Print(err.Error())
		sendError("Internal error", w, ServerError, 500)
		return
	}

	handleError(sendSuccess(w, sourceAddResponse{
		Status:   ResponseSuccessStr,
		Secret:   source.Secret,
		SourceID: source.SourceID,
	}), w, ServerError, 500)
}

func login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest

	if !handleUserInput(w, r, &request) {
		return
	}
	if isStructInvalid(request) || len(request.Password) != 128 {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	token, success, err := loginQuery(db, request.Username, request.Password)
	if err != nil {
		sendError("Internal error", w, ServerError, 500)
		return
	}

	if success {
		handleError(sendSuccess(w, loginResponse{
			Status: ResponseSuccessStr,
			Token:  token,
		}), w, ServerError, 500)
	} else {
		handleError(sendSuccess(w, loginResponse{
			Status: ResponseErrorStr,
		}), w, ServerError, 500)
	}
}

func handleUserInput(w http.ResponseWriter, r *http.Request, p interface{}) bool {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 10000))
	if err != nil {
		LogError("ReadError: " + err.Error())
		return false
	}
	if err := r.Body.Close(); err != nil {
		LogError("ReadError: " + err.Error())
		return false
	}

	errEncode := json.Unmarshal(body, p)
	if handleError(errEncode, w, WrongInputFormatError, 422) {
		return false
	}
	return true
}

func handleError(err error, w http.ResponseWriter, message ErrorMessage, statusCode int) bool {
	if err == nil {
		return false
	}
	sendError(err.Error(), w, message, statusCode)
	return true
}

func sendError(erre string, w http.ResponseWriter, message ErrorMessage, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if statusCode >= 500 {
		LogCritical(erre)
	} else {
		LogError(erre)
	}
	w.WriteHeader(statusCode)

	var de []byte
	var err error
	if len(string(message)) == 0 {
		de, err = json.Marshal(&ResponseError)
	} else {
		de, err = json.Marshal(&Status{"error", string(message)})
	}

	if err != nil {
		panic(err)
	}
	_, _ = fmt.Fprintln(w, string(de))
}

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

func sendSuccess(w http.ResponseWriter, i interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	de, err := json.Marshal(i)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(de))
	if err != nil {
		return err
	}
	return nil
}
