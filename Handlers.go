package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

var (
	db *dbhelper.DBhelper
)

//Subscriptions ------------------------------
//-> /sub/remove
func unsubscribe(w http.ResponseWriter, r *http.Request) {
	var request unsubscribeRequest
	if !parseUserInput(w, r, &request) {
		return
	}
	if len(request.SubscriptionID) != 32 {
		sendError("input missing wrong lengh", w, WrongInputFormatError, 422)
		return
	}
	err := removeSubscription(db, request.SubscriptionID)
	if err != nil {
		handleServerError(w, err)
		return
	}
	sendResponse(w, ResponseSuccess, "", nil)
	return
}

//-> /sub/add
func subscribe(w http.ResponseWriter, r *http.Request) {
	var request subscriptionRequest

	if !parseUserInput(w, r, &request) {
		return
	}
	token := request.Token
	if token == "-" {
		token = ""
	}
	if isStructInvalid(request) {
		sendError("input missing", w, InvalidTokenError, 422)
		return
	}
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, InvalidTokenError, 403)
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

	if userID > 1 {
		is, err := user.isSubscribedTo(db, source.PkID)
		if err != nil {
			sendError("input missing", w, ServerError, 500)
			fmt.Println(err.Error())
			return
		}
		if is {
			sendResponse(w, ResponseError, "You can only subscribe one time to a source", nil)
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
			handleServerError(w, err)
			return
		}
		response := subscriptionResponse{
			SubscriptionID: sub.SubscriptionID,
			Name:           source.Name,
		}
		sendResponse(w, ResponseSuccess, "", response)
	} else {
		sendResponse(w, ResponseError, ActionNotAllowed, nil)
	}
}

//Sources ------------------------------
//-> /source/add
func createSource(w http.ResponseWriter, r *http.Request) {
	var request sourceAddRequest
	if !parseUserInput(w, r, &request) {
		return
	}
	if isStructInvalid(request) {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	if len(request.Token) != 64 {
		sendError("token invalid", w, InvalidTokenError, 403)
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
		handleServerError(w, err)
		return
	}

	sendResponse(w, ResponseSuccess, "", sourceAddResponse{
		Secret:   source.Secret,
		SourceID: source.SourceID,
	})
}

//-> /sources
func listSources(w http.ResponseWriter, r *http.Request) {
	var request sourceRequest

	if !parseUserInput(w, r, &request) {
		return
	}

	if len(request.Token) != 64 {
		sendError("input missing wrong lengh", w, InvalidTokenError, 422)
		return
	}

	user, err := getUserIDFromSession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}
	var response listSourcesResponse
	if len(request.SourceID) == 0 {
		sources, err := getSourcesForUser(db, user.Pkid)
		if err != nil {
			handleServerError(w, err)
			return
		}

		response = listSourcesResponse{
			Sources: sources,
		}

	} else {
		source, err := getSourceFromSourceID(db, request.SourceID)
		if err != nil {
			handleServerError(w, err)
			return
		}
		if user.Pkid != source.CreatorID {
			source.CreatorID = 0
			source.Secret = ""
			if source.IsPrivate {
				source.Description = "This is a private source"
				source.Name = "Private"
			}
		}

		response = listSourcesResponse{
			Sources: []Source{*source},
		}
	}

	sendResponse(w, ResponseSuccess, "", response)
}

//-> /source/remove
func removeSource(w http.ResponseWriter, r *http.Request) {
	var request sourceRequest
	if !parseUserInput(w, r, &request) {
		return
	}
	if isStructInvalid(request) {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}
	if len(request.Token) != 64 {
		sendError("token invalid", w, InvalidTokenError, 403)
		return
	}

	user, err := getUserIDFromSession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}

	source, err := getSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("Server error", w, ServerError, 500)
		return
	}

	if source.CreatorID != user.Pkid {
		sendError("user not allowed", w, ActionNotAllowed, 403)
		return
	}

	err = deleteSource(db, source.PkID)

	if err != nil {
		handleServerError(w, err)
		return
	}

	sendResponse(w, ResponseSuccess, "", nil)
}

//User functions ------------------------------
//-> /login
func login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest

	if !parseUserInput(w, r, &request) {
		return
	}
	if isStructInvalid(request) || len(request.Password) != 128 {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	token, success, err := loginQuery(db, request.Username, request.Password)
	if err != nil {
		handleServerError(w, err)
		return
	}

	if success {
		sendResponse(w, ResponseSuccess, "", loginResponse{
			Token: token,
		})
	} else {
		sendResponse(w, ResponseError, "Error logging in", nil, 403)
	}
}

func sendResponse(w http.ResponseWriter, status ResponseStatus, message string, payload interface{}, params ...int) error {
	statusCode := http.StatusOK
	s := "0"
	if status == 1 {
		s = "1"
	}

	w.Header().Set(HeaderStatus, s)
	w.Header().Set(HeaderStatusMessage, message)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if len(params) > 0 {
		statusCode = params[0]
		w.WriteHeader(statusCode)
	}

	toSend := "error"
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		toSend = string(b)
	}

	_, err := fmt.Fprintln(w, toSend)
	return err
}

//rest functions
func parseUserInput(w http.ResponseWriter, r *http.Request, p interface{}) bool {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 100000))
	if err != nil {
		LogError("ReadError: " + err.Error())
		return false
	}
	if err := r.Body.Close(); err != nil {
		LogError("ReadError: " + err.Error())
		return false
	}

	return !handleError(json.Unmarshal(body, p), w, WrongInputFormatError, 422)
}

func handleError(err error, w http.ResponseWriter, message string, statusCode int) bool {
	if err == nil {
		return false
	}
	sendError(err.Error(), w, message, statusCode)
	return true
}

func sendError(erre string, w http.ResponseWriter, message string, statusCode int) {
	if statusCode >= 500 {
		LogCritical(erre)
	} else {
		LogError(erre)
	}
	sendResponse(w, ResponseError, message, nil, statusCode)
}

func handleServerError(w http.ResponseWriter, err error) {
	sendError("internal server error", w, ServerError, 500)
	if err != nil {
		log.Println(err.Error())
	}
}
