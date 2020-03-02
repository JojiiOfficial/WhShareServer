package handlers

import (
	"net/http"

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
		if err.Error() == dbhelper.ErrNoRowsInResultSet {
			sendResponse(w, models.ResponseSuccess, "", nil)
			return
		}

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

	if len(request.SubscriptionID) != 32 {
		sendResponse(w, models.ResponseError, "Invalid subscriptionID length!", nil, http.StatusUnprocessableEntity)
		return
	}

	if checkPayloadSizes(w, constants.DefaultMaxPayloadSize, request.CallbackURL) {
		return
	}

	// Ignore if user is admin, otherwise validate callback url
	if !handler.user.IsAdmin() && !validateCallbackURL(handler.config, w, request.CallbackURL, genIPBlocklist(handler.ownIP, handler.config)) {
		return
	}

	subscription, err := models.GetSubscriptionBySubsID(db, request.SubscriptionID)
	if err != nil {
		sendServerError(w)
		return
	}

	//Update only if it's users source or user not logged in and sourceID matches
	if (handler.user != nil && subscription.UserID == handler.user.Pkid) || handler.user == nil {
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

	if checkInput(w, request, request.CallbackURL) {
		return
	}

	if len(request.SourceID) != 32 {
		sendResponse(w, models.ResponseError, models.WrongLength, nil, http.StatusUnprocessableEntity)
		return
	}

	//If client is logged in and no admin
	if handler.user != nil && !handler.user.IsAdmin() {
		//Check if user can subscribe to sources
		if !handler.user.CanSubscribe() {
			sendResponse(w, models.ResponseError, "You are not allowed to have subscriptions", nil, http.StatusForbidden)
			return
		}

		//Check if user subscription limit is exceeded
		isLimitReached, err := handler.user.IsSubscriptionLimitReached(db)
		if err != nil {
			sendServerError(w)
			return
		}

		if isLimitReached {
			sendResponse(w, models.ResponseError, "Subscription limit exceeded", nil, http.StatusForbidden)
			return
		}
	}

	// Ignore if user is admin, otherwise validate callback url
	if !handler.user.IsAdmin() && !validateCallbackURL(handler.config, w, request.CallbackURL, genIPBlocklist(handler.ownIP, handler.config)) {
		return
	}

	//The source to get subbed
	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		sendError("input missing", w, models.NotFoundError, http.StatusNotFound)
		return
	}

	var isSubscribed bool
	if handler.user != nil {
		//Check if user already has subscripted to this source
		isSubscribed, err = handler.user.IsSubscribedTo(db, source.PkID)
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
	if source.IsPrivate && handler.user != nil && source.CreatorID == handler.user.Pkid || !source.IsPrivate {
		uID := uint32(1)
		if handler.user != nil {
			uID = handler.user.Pkid
		}

		subs := models.Subscription{
			Source:      source.PkID,
			CallbackURL: request.CallbackURL,
			UserID:      uID,
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
