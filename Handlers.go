package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/gorilla/mux"
)

const defaultMaxPayloadSize = uint(150)

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
		sendServerError(w)
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

	//Validate callbackURL -> returns error if invalid
	isReserved, err := gaw.IsReserved(request.CallbackURL)
	if err != nil {
		sendResponse(w, ResponseError, InvalidCallbackURL, 406)
		return
	}

	//Check if ip is bogon IPs are allowed. If not check IP
	if !config.Server.BogonAsCallback && isReserved {
		sendError("ip reserved", w, "CallbackURL points to reserved IP", 422)
		return
	}

	//Determine the user
	userID := uint32(1)
	var user *User
	if len(token) > 0 {
		user, err = getUserIDFromSession(db, token)
		if err != nil {
			log.Error(err)
			userID = 1
		} else {
			userID = user.Pkid
			user.updateIP(db, gaw.GetIPFromHTTPrequest(r))
		}
	}

	//The source to get subbed
	source, err := getSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, NotFoundError, 422)
		return
	}

	var isSubscribed bool
	if userID > 1 {
		is, err := user.isSubscribedTo(db, source.PkID)
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

	if checkInput(w, request, request.Token, request.Name, request.Token) {
		return
	}

	user, err := getUserIDFromSession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}
	user.updateIP(db, gaw.GetIPFromHTTPrequest(r))

	nameExitst, err := user.hasSourceWithName(db, request.Name)
	if err != nil {
		sendServerError(w)
		return
	}

	if nameExitst {
		sendResponse(w, ResponseError, MultipleSourceNameErr, nil)
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

	user, err := getUserIDFromSession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}
	user.updateIP(db, gaw.GetIPFromHTTPrequest(r))

	var response listSourcesResponse
	if len(request.SourceID) == 0 {
		sources, err := getSourcesForUser(db, user.Pkid)
		if err != nil {
			sendServerError(w)
			return
		}

		response = listSourcesResponse{
			Sources: sources,
		}

	} else {
		source, err := getSourceFromSourceID(db, request.SourceID)
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
			Sources: []Source{*source},
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

	user, err := getUserIDFromSession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, InvalidTokenError, 403)
		return
	}

	user.updateIP(db, gaw.GetIPFromHTTPrequest(r))

	source, err := getSourceFromSourceID(db, request.SourceID)
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
			err = source.delete(db)
		}
	case actions[1]:
		{
			//change description
			err = source.update(db, "description", request.Content, true)
		}
	case actions[2]:
		{
			//rename
			has, err := user.hasSourceWithName(db, request.Content)
			if err != nil {
				sendServerError(w)
				return
			}

			if has {
				sendResponse(w, ResponseError, MultipleSourceNameErr, nil)
				return
			}
			err = source.update(db, "name", request.Content)
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
			err = source.update(db, "private", newVal)
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

	token, success, err := loginQuery(db, request.Username, gaw.SHA512(request.Password+request.Username), gaw.GetIPFromHTTPrequest(r))
	if err != nil {
		sendServerError(w)
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

	exists, err := userExitst(db, request.Username)
	if err != nil {
		sendServerError(w)
		return
	}

	if exists {
		sendResponse(w, ResponseError, "User exists", nil)
		return
	}

	err = insertUser(db, request.Username, gaw.SHA512(request.Password+request.Username), gaw.GetIPFromHTTPrequest(r))
	if err != nil {
		sendServerError(w)
		return
	}

	sendResponse(w, ResponseSuccess, "", nil)
}

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

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceID := vars["sourceID"]
	secret := vars["secret"]

	if len(sourceID) == 0 || len(secret) == 0 {
		log.Info("source or secret is not given in webhook!")
		return
	}

	source, err := getSourceFromSourceID(db, sourceID)
	if err != nil {
		log.Info("webhookHandler - Source not found")
		sendResponse(w, ResponseError, "404 Not found", nil, 404)
		return
	}

	if source.Secret == secret {
		c := make(chan bool, 1)
		log.Info("New valid webhook:", source.Name)

		go (func(req *http.Request) {
			//Don't forward the webhook if it contains a header-value pair which is on the blacklist
			if isHeaderBlocklistetd(req.Header, &config.Server.WebhookBlacklist.HeaderValues) {
				log.Warnf("Blocked webhook '%s' because of header-blacklist\n", source.SourceID)

				c <- true
				return
			}

			//Read payload body from webhook
			payload, err := ioutil.ReadAll(io.LimitReader(req.Body, 100000))
			if err != nil {
				LogError(err)
				c <- false
				return
			}
			req.Body.Close()

			//Delete in config specified json objects
			payload, err = gaw.JSONRemoveItems(payload, config.Server.WebhookBlacklist.JSONObjects[ModeToString[source.Mode]], false)
			if err != nil {
				LogError(err, log.Fields{
					"msg": "Error filtering JSON!",
				})

				c <- true
				return
			}

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
		log.Warn("invalid secret for source", sourceID)
	}
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
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 100000))

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
