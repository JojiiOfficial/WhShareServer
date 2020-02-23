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
func Unsubscribe(db *dbhelper.DBhelper, config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) {
	var request models.UnsubscribeRequest
	if !parseUserInput(config, w, r, &request) {
		return
	}
	if len(request.SubscriptionID) != 32 {
		sendError("input missing wrong length", w, models.WrongInputFormatError, 422)
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
	sendResponse(w, models.ResponseSuccess, "", nil)
	return
}

//UpdateCallbackURL update subscription url handler
//-> /sub/updateCallbackURL
func UpdateCallbackURL(db *dbhelper.DBhelper, config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) {
	var request models.SubscriptionUpdateCallbackRequest
	if !parseUserInput(config, w, r, &request) {
		return
	}

	token := request.Token
	if token == "-" {
		token = ""
	}

	//Check if token available. return error if not valid, but given.
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, models.InvalidTokenError, 403)
		return
	}

	if len(request.SubscriptionID) != 32 {
		sendResponse(w, models.ResponseError, "Invalid subscriptionID length!", nil, 411)
		return
	}

	if checkPayloadSizes(w, constants.DefaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	//TODO GET IP
	if !validateCallbackURL(config, w, request.CallbackURL, "") {
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
func Subscribe(db *dbhelper.DBhelper, config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) {
	var request models.SubscriptionRequest

	if !parseUserInput(config, w, r, &request) {
		return
	}

	token := request.Token
	if token == "-" {
		token = ""
	}

	if isStructInvalid(request) {
		sendError("input missing", w, models.InvalidTokenError, 422)
		return
	}

	if checkPayloadSizes(w, constants.DefaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	//Check if token available. return error if not valid, but given.
	if len(token) > 0 && len(token) != 64 {
		sendError("token invalid", w, models.InvalidTokenError, 403)
		return
	}

	if len(request.SourceID) != 32 {
		sendResponse(w, models.ResponseError, models.WrongLength, nil, 411)
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
			sendResponse(w, models.ResponseError, "You are not allowed to have subscriptions", nil, 403)
			return
		} else if user.Role.MaxSubscriptions != -1 && userSubscriptions >= uint32(user.Role.MaxSubscriptions) {
			sendResponse(w, models.ResponseError, "Subscription limit exceeded", nil, 403)
			return
		}

		userID = user.Pkid
	}

	//TODO get own IP
	if !validateCallbackURL(config, w, request.CallbackURL, "") {
		return
	}

	//The source to get subbed
	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, models.NotFoundError, 422)
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
		ex, err := models.SubscriptionExists(db, source.PkID, request.CallbackURL)
		if err != nil {
			sendServerError(w)
			return
		}
		isSubscribed = ex
	}

	if isSubscribed {
		sendResponse(w, models.ResponseError, "You can subscribe to a source only once", nil)
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
