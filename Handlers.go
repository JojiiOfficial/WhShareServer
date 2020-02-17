package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/gorilla/mux"
)

//Subscriptions ------------------------------
//-> /sub/remove
func unsubscribe(w http.ResponseWriter, r *http.Request) {
	var request unsubscribeRequest
	if !parseUserInput(w, r, &request) {
		return
	}
	if len(request.SubscriptionID) != 32 {
		sendError("input missing wrong length", w, WrongInputFormatError, 422)
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

	if gaw.IsReserved(request.CallbackURL) && !config.Server.BogonAsCallback {
		sendError("ip reserved", w, "CallbackURL points to reserved IP", 422)
		return
	}

	//Determine the user
	userID := uint32(1)
	var user *User
	var err error
	if len(token) > 0 {
		user, err = getUserIDFromSession(db, token)
		if err != nil {
			log.Println(err.Error())
			userID = 1
		} else {
			userID = user.Pkid
		}
	}

	//The source to get subbed
	source, err := getSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, NotFoundError, 422)
		log.Println(err.Error())
		return
	}

	var isSubscribed bool
	if userID > 1 {
		is, err := user.isSubscribedTo(db, source.PkID)
		if err != nil {
			handleServerError(w, err)
			return
		}
		isSubscribed = is
	} else {
		ex, err := checkSubscriptionExitsts(db, source.PkID, request.CallbackURL)
		if err != nil {
			handleServerError(w, err)
			return
		}
		isSubscribed = ex
	}

	if isSubscribed {
		sendResponse(w, ResponseError, "You can only subscribe one time to a source", nil)
		return
	}

	if source.IsPrivate && source.CreatorID == userID || !source.IsPrivate {
		subs := Subscription{
			Source:      source.PkID,
			CallbackURL: request.CallbackURL,
			UserID:      userID,
		}

		err := subs.insert(db)
		if err != nil {
			log.Println(err.Error())
			handleServerError(w, err)
			return
		}

		response := subscriptionResponse{
			SubscriptionID: subs.SubscriptionID,
			Name:           source.Name,
			Mode:           source.Mode,
		}

		sendResponse(w, ResponseSuccess, "", response)
		go subs.startValidation(source.SourceID)
	} else {
		sendResponse(w, ResponseError, ActionNotAllowed, nil)
	}
}

//Sources ------------------------------
//-> /source/create
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
		Mode:        request.Mode,
		Description: request.Description,
	}

	err = source.insert(db)
	if err != nil {
		log.Print(err.Error())
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
		sendError("input missing wrong length", w, InvalidTokenError, 422)
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

	err = source.delete(db)

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

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceID := vars["sourceID"]
	secret := vars["secret"]

	if len(sourceID) == 0 || len(secret) == 0 {
		log.Println("source or secret is not given in webhook!")
		return
	}

	source, err := getSourceFromSourceID(db, sourceID)
	if err != nil {
		sendResponse(w, ResponseError, "404 Not found", nil, 404)
		return
	}
	c := make(chan bool, 1)
	if source.Secret == secret {
		log.Println("New valid webhook:", source.Name)

		go (func(req *http.Request) {
			//Read payload body from webhook
			payload, err := ioutil.ReadAll(io.LimitReader(req.Body, 100000))
			if err != nil {
				log.Println("ReadError: " + err.Error())
				c <- false
				return
			}
			req.Body.Close()
			c <- true

			headers := headerToString(req.Header)

			webhook := &Webhook{
				SourceID: source.PkID,
				Headers:  headers,
				Payload:  string(payload),
			}
			webhook.insert(db)
			notifyAllSubscriber(db, webhook, source)
		})(r)

		if <-c {
			sendResponse(w, ResponseSuccess, "success", nil)
		} else {
			sendResponse(w, ResponseError, "error", nil, 500)
		}

	} else {
		log.Println("invalid secret for source", sourceID)
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
	} else if len(message) > 0 {
		toSend = message
	}

	_, err := fmt.Fprintln(w, toSend)
	return err
}

//rest functions
func parseUserInput(w http.ResponseWriter, r *http.Request, p interface{}) bool {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 100000))
	if err != nil {
		log.Println("ReadError: " + err.Error())
		return false
	}
	if err := r.Body.Close(); err != nil {
		log.Println("ReadError: " + err.Error())
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
		log.Println(erre)
	} else {
		log.Println(erre)
	}
	sendResponse(w, ResponseError, message, nil, statusCode)
}

func handleServerError(w http.ResponseWriter, err error) {
	sendError("internal server error", w, ServerError, 500)
	if err != nil {
		log.Println(err.Error())
	}
}
