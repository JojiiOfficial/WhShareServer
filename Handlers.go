package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/constants"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/gorilla/mux"
)

const defaultMaxPayloadSize = uint(150)

//TODO move to new package `functions`

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

	subscription, err := models.GetSubscriptionBySubsID(db, request.SubscriptionID)
	if err != nil {
		sendServerError(w)
		return
	}

	err = subscription.Remove(db)
	if err != nil {
		sendServerError(w)
		return
	}
	sendResponse(w, ResponseSuccess, "", nil)
	return
}

//-> /sub/updateCallbackURL
func updateCallbackURL(w http.ResponseWriter, r *http.Request) {
	var request subscriptionUpdateCallbackRequest
	if !parseUserInput(w, r, &request) {
		return
	}

	token := request.Token
	if token == "-" {
		token = ""
	}

	//Check if token available. return error if not valid, but given.
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, InvalidTokenError, 403)
		return
	}

	if len(request.SubscriptionID) != 32 {
		sendResponse(w, ResponseError, "Invalid subscriptionID length!", nil, 411)
		return
	}

	if checkPayloadSizes(w, defaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	if !validateCallbackURL(w, request.CallbackURL) {
		return
	}

	//Determine the user
	userID := uint32(1)
	if len(token) > 0 {
		user, err := models.GetUserBySession(db, token)
		if err != nil {
			LogError(err)
			userID = 1
		} else {
			userID = user.Pkid
			go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))
		}
	}

	subscription, err := models.GetSubscriptionBySubsID(db, request.SubscriptionID)
	if err != nil {
		sendServerError(w)
		return
	}

	//Update only if it's users source or user not logged in and sourceID matches
	if (userID > 1 && subscription.UserID == userID) || subscription.UserID == 1 {
		err = subscription.UpdateCallback(db, request.CallbackURL)
		if err != nil {
			sendServerError(w)
		} else {
			sendResponse(w, ResponseSuccess, "", nil)
		}
	} else {
		sendResponse(w, ResponseError, ActionNotAllowed, nil)
	}
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

	if checkPayloadSizes(w, defaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	//Check if token available. return error if not valid, but given.
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, InvalidTokenError, 403)
		return
	}

	if len(request.SourceID) != 32 {
		sendResponse(w, ResponseError, WrongLength, nil, 411)
		return
	}

	//Determine the user
	userID := uint32(1)
	var user *models.User
	var err error
	if len(token) > 0 {
		user, err = models.GetUserBySession(db, token)
		if err != nil {
			sendServerError(w)
			return

		}
		go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))
		userSubscriptions, err := user.GetSubscriptionCount(db)
		if err != nil {
			sendServerError(w)
			return
		}

		if user.Role.MaxSubscriptions == 0 {
			sendResponse(w, ResponseError, "You are not allowed to have subscriptions", nil, 403)
			return
		} else if user.Role.MaxSubscriptions != -1 && userSubscriptions >= uint32(user.Role.MaxSubscriptions) {
			sendResponse(w, ResponseError, "Subscription limit exceeded", nil, 403)
			return
		}

		userID = user.Pkid
	}

	if !validateCallbackURL(w, request.CallbackURL) {
		return
	}

	//The source to get subbed
	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, NotFoundError, 422)
		return
	}

	var isSubscribed bool
	if userID > 1 {
		is, err := user.IsSubscribedTo(db, source.PkID)
		if err != nil {
			sendServerError(w)
			return
		}
		isSubscribed = is
	} else {
		ex, err := checkSubscriptionExitsts(db, source.PkID, request.CallbackURL)
		if err != nil {
			sendServerError(w)
			return
		}
		isSubscribed = ex
	}

	if isSubscribed {
		sendResponse(w, ResponseError, "You can subscribe to a source only once", nil)
		return
	}

	if source.IsPrivate && source.CreatorID == userID || !source.IsPrivate {
		subs := models.Subscription{
			Source:      source.PkID,
			CallbackURL: request.CallbackURL,
			UserID:      userID,
		}

		err := subs.Insert(db)
		if err != nil {
			sendServerError(w)
			return
		}

		response := subscriptionResponse{
			SubscriptionID: subs.SubscriptionID,
			Name:           source.Name,
			Mode:           source.Mode,
		}

		sendResponse(w, ResponseSuccess, "", response)
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

	if checkInput(w, request, request.Token, request.Name) {
		return
	}

	user, err := models.GetUserBySession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}

	go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

	//Check if user is allowed to create sources
	if request.Private && user.Role.MaxPrivSources == 0 {
		sendResponse(w, ResponseError, "You are not allowed to have private sources", nil, 403)
		return
	} else if !request.Private && user.Role.MaxPubSources == 0 {
		sendResponse(w, ResponseError, "You are not allowed to have public sources", nil, 403)
		return
	}

	scount, err := user.GetSourceCount(db, request.Private)
	if err != nil {
		sendServerError(w)
		return
	}

	//Check for source limit
	if request.Private && user.Role.MaxPrivSources != -1 {
		if scount >= uint(user.Role.MaxPrivSources) {
			sendResponse(w, ResponseError, "Limit for private sources exceeded", nil, 403)
			return
		}
	} else if !request.Private && user.Role.MaxPubSources != -1 {
		if scount >= uint(user.Role.MaxPubSources) {
			sendResponse(w, ResponseError, "Limit for public sources exceeded", nil, 403)
			return
		}
	}

	//Check if user already has a source with this name
	nameExitst, err := user.HasSourceWithName(db, request.Name)
	if err != nil {
		sendServerError(w)
		return
	}

	if nameExitst {
		sendResponse(w, ResponseError, MultipleSourceNameErr, nil)
		return
	}

	source := &models.Source{
		Creator:     *user,
		IsPrivate:   request.Private,
		Name:        request.Name,
		Mode:        request.Mode,
		Description: request.Description,
	}

	err = source.Insert(db)
	if err != nil {
		sendServerError(w)
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

	if checkInput(w, request, request.Token) {
		return
	}

	if request.SourceID == "-" {
		request.SourceID = ""
	}

	user, err := models.GetUserBySession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}

	go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

	var response listSourcesResponse
	if len(request.SourceID) == 0 {
		sources, err := models.GetSourcesForUser(db, user.Pkid)
		if err != nil {
			sendServerError(w)
			return
		}

		response = listSourcesResponse{
			Sources: sources,
		}

	} else {
		source, err := models.GetSourceFromSourceID(db, request.SourceID)
		if err != nil {
			sendServerError(w)
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
			Sources: []models.Source{*source},
		}
	}

	sendResponse(w, ResponseSuccess, "", response)
}

//-> /source/update/{action}
func updateSource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	action := vars["action"]
	actions := []string{
		"delete",
		"changedescr",
		"rename",
		"toggleAccess",
	}

	if !gaw.IsInStringArray(action, actions) {
		sendError("not available", w, WrongInputFormatError, 501)
		return
	}

	var request sourceRequest
	if !parseUserInput(w, r, &request) {
		return
	}

	if checkInput(w, request, request.Token, request.Content) {
		return
	}

	user, err := models.GetUserBySession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}

	go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		if err.Error() == dbhelper.ErrNoRowsInResultSet {
			//Source just not found if no rows in result set
			sendResponse(w, ResponseError, NotFoundError, nil, 404)
			return
		}
		//Real db error
		sendServerError(w)
		return
	}

	if source.CreatorID != user.Pkid {
		sendError("user not allowed", w, ActionNotAllowed, 403)
		return
	}

	err = nil
	message := ""
	switch action {
	case actions[0]:
		{
			//delete
			err = source.Delete(db)
		}
	case actions[1]:
		{
			//change description
			err = source.Update(db, "description", request.Content, true)
		}
	case actions[2]:
		{
			//rename
			has, err := user.HasSourceWithName(db, request.Content)
			if err != nil {
				sendServerError(w)
				return
			}

			if has {
				sendResponse(w, ResponseError, MultipleSourceNameErr, nil)
				return
			}
			err = source.Update(db, "name", request.Content)
		}
	case actions[3]:
		{
			//Toggle accessMode
			newVal := "1"
			message = "private"
			if source.IsPrivate {
				newVal = "0"
				message = "public"
			}
			err = source.Update(db, "private", newVal)
		}
	}

	if err != nil {
		sendServerError(w)
	} else {
		sendResponse(w, ResponseSuccess, message, nil)
	}
}

//User functions ------------------------------
//-> /user/login
func login(w http.ResponseWriter, r *http.Request) {
	var request credentialRequest

	if !parseUserInput(w, r, &request) {
		return
	}
	if isStructInvalid(request) || len(request.Password) != 128 {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	if checkPayloadSizes(w, defaultMaxPayloadSize, request.Username) {
		return
	}

	//Make the request take 1500ms
	after := time.After(1500 * time.Millisecond)

	token, success, err := models.LoginQuery(db, request.Username, gaw.SHA512(request.Password+request.Username), gaw.GetIPFromHTTPrequest(r))
	if err != nil {
		sendServerError(w)
		return
	}

	<-after

	if success {
		sendResponse(w, ResponseSuccess, "", loginResponse{
			Token: token,
		})
	} else {
		sendResponse(w, ResponseError, "Error logging in", nil, 403)
	}
}

//-> /user/create
func register(w http.ResponseWriter, r *http.Request) {
	if !config.Server.AllowRegistration {
		sendResponse(w, ResponseError, "Server doesn't accept registrations", nil, 403)
		return
	}

	var request credentialRequest

	if !parseUserInput(w, r, &request) {
		return
	}

	if isStructInvalid(request) || len(request.Password) != 128 || len(request.Username) > 30 {
		sendError("input missing", w, WrongInputFormatError, 422)
		return
	}

	exists, err := models.UserExists(db, request.Username)
	if err != nil {
		sendServerError(w)
		return
	}

	if exists {
		sendResponse(w, ResponseError, "User exists", nil)
		return
	}

	err = models.InsertUser(db, request.Username, gaw.SHA512(request.Password+request.Username), gaw.GetIPFromHTTPrequest(r))
	if err != nil {
		sendServerError(w)
		return
	}

	sendResponse(w, ResponseSuccess, "", nil)
}

//-> /get/webhook
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceID := vars["sourceID"]
	secret := vars["secret"]

	if len(sourceID) == 0 || len(secret) == 0 {
		log.Info("source or secret is not given in webhook!")
		return
	}

	source, err := models.GetSourceFromSourceID(db, sourceID)
	if err != nil {
		log.Warn("WebhookHandler - Source not found")
		sendResponse(w, ResponseError, "404 Not found", nil, 404)
		return
	}

	if source.Secret == secret {
		c := make(chan bool, 1)
		log.Info("New webhook:", source.Name)
		msg := "error"

		go (func(req *http.Request) {
			userChan := make(chan *models.User, 1)
			//Get source user
			go (func() {
				us, err := models.GetUserByPK(db, source.CreatorID)
				if err != nil {
					LogError(err)
					userChan <- nil
				} else {
					userChan <- us
				}
			})()

			//Don't forward the webhook if it contains a header-value pair which is on the blacklist
			if isHeaderBlocklistetd(req.Header, &config.Server.WebhookBlacklist.HeaderValues) {
				log.Warnf("Blocked webhook '%s' because of header-blacklist\n", source.SourceID)

				msg = "error"
				c <- true
				return
			}

			//Await getting user
			user := <-userChan

			//return on error or user not allowed to send hooks
			if user == nil || user.Role.MaxTraffic == 0 || user.Role.MaxHookCalls == 0 {
				c <- false
				msg = "not allowed to send hooks"
				return
			}

			//Read payload body from webhook
			payload, err := ioutil.ReadAll(io.LimitReader(req.Body, 100000))
			if err != nil {
				LogError(err)
				msg = "error reading content"
				c <- false
				return
			}
			req.Body.Close()

			headers := headerToString(req.Header)

			//Calculate traffic of request
			reqTraffic := uint32(len(payload)) + uint32(len(headers))

			//Check if user limit exceeded
			if (user.Role.MaxTraffic != -1 && uint32(user.Role.MaxTraffic*1024) <= (user.Traffic+reqTraffic)) ||
				(user.Role.MaxHookCalls != -1 && user.Role.MaxHookCalls < int(user.HookCalls+1)) {
				msg = "traffic/hookCall limit exceeded"
				c <- false
				return
			}

			//Delete in config specified json objects
			payload, err = gaw.JSONRemoveItems(payload, config.Server.WebhookBlacklist.JSONObjects[constants.ModeToString[source.Mode]], false)
			if err != nil {
				LogError(err, log.Fields{
					"msg": "Error filtering JSON!",
				})
				msg = "server error"
				c <- true
				return
			}

			c <- true

			//Update traffic and hookCallCount if not both unlimited
			if user.Role.MaxHookCalls != -1 || user.Role.MaxTraffic != -1 {
				user.AddHookCall(db, reqTraffic)
			}

			webhook := &models.Webhook{
				SourceID: source.PkID,
				Headers:  headers,
				Payload:  string(payload),
			}
			webhook.Insert(db)
			models.NotifyAllSubscriber(db, config, webhook, source, subCB{retryService: retryService})
		})(r)

		if <-c {
			sendResponse(w, ResponseSuccess, "Success", nil)
		} else {
			sendResponse(w, ResponseError, msg, nil, 500)
		}

	} else {
		log.Warn("invalid secret for source", sourceID)
	}
}

// ------------------------ REST Helper functions -----------------------

//Returns true on error
func checkInput(w http.ResponseWriter, request interface{}, token string, contents ...string) bool {
	if isStructInvalid(request) {
		sendError("input missing", w, WrongInputFormatError, 422)
		return true
	}

	if len(token) != 64 {
		sendError("token invalid", w, InvalidTokenError, 403)
		return true
	}

	return checkPayloadSizes(w, defaultMaxPayloadSize, contents...)
}

//Returns true on error
func checkPayloadSizes(w http.ResponseWriter, maxPayloadSize uint, contents ...string) bool {
	for _, content := range contents {
		if uint(len(content)) > maxPayloadSize-1 {
			sendResponse(w, ResponseError, "Content too long!", nil, 413)
			return true
		}
	}
	return false
}

func sendResponse(w http.ResponseWriter, status ResponseStatus, message string, payload interface{}, params ...int) {
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

	var err error
	if payload != nil {
		err = json.NewEncoder(w).Encode(payload)
	} else if len(message) > 0 {
		_, err = fmt.Fprintln(w, message)
	}

	LogError(err)
}

//parseUserInput tries to read the body and parse it into p. Returns true on success
func parseUserInput(w http.ResponseWriter, r *http.Request, p interface{}) bool {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, config.Webserver.MaxBodyLength))

	if LogError(err) || LogError(r.Body.Close()) {
		return false
	}

	return !handleAndSendError(json.Unmarshal(body, p), w, WrongInputFormatError, 422)
}

func handleAndSendError(err error, w http.ResponseWriter, message string, statusCode int) bool {
	if !LogError(err) {
		return false
	}
	sendError(err.Error(), w, message, statusCode)
	return true
}

func sendError(erre string, w http.ResponseWriter, message string, statusCode int) {
	sendResponse(w, ResponseError, message, nil, statusCode)
}

func sendServerError(w http.ResponseWriter) {
	sendError("internal server error", w, ServerError, 500)
}

//Return true if exit
func validateCallbackURL(w http.ResponseWriter, callbackURL string) bool {
	//set addIP to server IP if serverHost as callback is disabled
	addIP := ""
	if !config.Server.ServerHostAsCallback {
		addIP = currIP
	}
	//Check if ip is bogon IPs are allowed. If not check IP
	isCallbackValid, err := isValidCallback(callbackURL, config.Server.BogonAsCallback, addIP)
	if LogError(err) {
		sendServerError(w)
		return false
	}

	if !isCallbackValid {
		sendError("ip reserved", w, "CallbackURL points to reserved IP, is Servers IP or can't lookup host", 422)
		return false
	}

	return true
}
