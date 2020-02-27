package handlers

import (
	"net/http"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/constants"
	"github.com/JojiiOfficial/WhShareServer/models"
)

//Unsubscribe unsubscribe handler
//-> /sub/remove
func Unsubscribe(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
	var request models.UnsubscribeRequest
	if !parseUserInput(handlerData.config, w, r, &request) {
		return
	}
	if len(request.SubscriptionID) != 32 {
		sendError("input missing wrong length", w, models.WrongInputFormatError, 422)
		return
	}

	subscription, err := models.GetSubscriptionBySubsID(db, request.SubscriptionID)
	if err != nil {
		//TODO send subscription not found
		sendServerError(w)
		return
	}

	err = subscription.Remove(db)
	if err != nil {
		sendServerError(w)
		return
	}
	sendResponse(w, models.ResponseSuccess, "", nil)
	return
}

//UpdateCallbackURL update subscription url handler
//-> /sub/updateCallbackURL
func UpdateCallbackURL(db *dbhelper.DBhelper, handler handlerData, w http.ResponseWriter, r *http.Request) {
	var request models.SubscriptionUpdateCallbackRequest
	if !parseUserInput(handler.config, w, r, &request) {
		return
	}

	token := request.Token
	if token == "-" {
		token = ""
	}

	//Check if token available. return error if not valid, but given.
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, models.InvalidTokenError, http.StatusForbidden)
		return
	}

	if len(request.SubscriptionID) != 32 {
		sendResponse(w, models.ResponseError, "Invalid subscriptionID length!", nil, http.StatusUnprocessableEntity)
		return
	}

	if checkPayloadSizes(w, constants.DefaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	if !validateCallbackURL(handler.config, w, request.CallbackURL, genIPBlocklist(handler.ownIP, handler.config)) {
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
			sendResponse(w, models.ResponseSuccess, "", nil)
		}
	} else {
		sendResponse(w, models.ResponseError, models.ActionNotAllowed, nil)
	}
}

//Subscribe subscription handler
//-> /sub/add
func Subscribe(db *dbhelper.DBhelper, handler handlerData, w http.ResponseWriter, r *http.Request) {
	var request models.SubscriptionRequest

	if !parseUserInput(handler.config, w, r, &request) {
		return
	}

	token := request.Token
	if token == "-" {
		token = ""
	}

	if isStructInvalid(request) {
		sendError("input missing", w, models.InvalidTokenError, http.StatusUnprocessableEntity)
		return
	}

	if checkPayloadSizes(w, constants.DefaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	//Check if token available. return error if not valid, but given.
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, models.InvalidTokenError, http.StatusForbidden)
		return
	}

	if len(request.SourceID) != 32 {
		sendResponse(w, models.ResponseError, models.WrongLength, nil, http.StatusUnprocessableEntity)
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

		//Check if user can subscribe to sources
		if !user.CanSubscribe() {
			sendResponse(w, models.ResponseError, "You are not allowed to have subscriptions", nil, http.StatusForbidden)
			return
		}

		//Check if user subscription limit is exceeded
		isLimitReached, err := user.IsSubscriptionLimitReached(db)
		if err != nil {
			sendServerError(w)
			return
		}

		if isLimitReached {
			sendResponse(w, models.ResponseError, "Subscription limit exceeded", nil, http.StatusForbidden)
			return
		}

		userID = user.Pkid
	}

	//Return if callback is invalid
	if !validateCallbackURL(handler.config, w, request.CallbackURL, genIPBlocklist(handler.ownIP, handler.config)) {
		return
	}

	//The source to get subbed
	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, models.NotFoundError, http.StatusNotFound)
		return
	}

	var isSubscribed bool
	if userID > 1 {
		//Check if user already has subscripted to this source
		isSubscribed, err = user.IsSubscribedTo(db, source.PkID)
		if err != nil {
			sendServerError(w)
			return
		}
	} else {
		//Check if subscription exists by comparing callback url and source
		isSubscribed, err = models.SubscriptionExists(db, source.PkID, request.CallbackURL)
		if err != nil {
			sendServerError(w)
			return
		}
	}

	//Return if already subscribed
	if isSubscribed {
		sendResponse(w, models.ResponseError, "You can subscribe to a source only once", nil)
		return
	}

	//Check if user is allowed to subscribe to the given source
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

		response := models.SubscriptionResponse{
			SubscriptionID: subs.SubscriptionID,
			Name:           source.Name,
			Mode:           source.Mode,
		}

		sendResponse(w, models.ResponseSuccess, "", response)
	} else {
		sendResponse(w, models.ResponseError, models.ActionNotAllowed, nil)
	}
}

//Get list of disallowed IPs
func genIPBlocklist(ownIP *string, config *models.ConfigStruct) []string {
	var list []string

	if ownIP != nil {
		list = append(list, *ownIP)
	}
	list = append(list, config.Server.BlocklistIPs...)

	return list
}

//Return true if exit
func validateCallbackURL(config *models.ConfigStruct, w http.ResponseWriter, callbackURL string, balckListedIPs []string) bool {
	//Check if ip is bogon IPs are allowed. If not check IP
	isCallbackValid, err := isValidCallback(callbackURL, config.Server.BogonAsCallback, balckListedIPs...)
	if LogError(err) {
		sendServerError(w)
		return false
	}

	if !isCallbackValid {
		sendError("ip reserved", w, "CallbackURL points to reserved IP, is Servers IP or can't lookup host", http.StatusForbidden)
		return false
	}

	return true
}
