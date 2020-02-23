package handlers

import (
	"net/http"
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/constants"
	"github.com/JojiiOfficial/WhShareServer/models"
)

//Login login handler
//-> /user/login
func Login(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
	var request models.CredentialRequest

	if !parseUserInput(handlerData.config, w, r, &request) {
		return
	}
	if isStructInvalid(request) || len(request.Password) != 128 {
		sendError("input missing", w, models.WrongInputFormatError, 422)
		return
	}

	if checkPayloadSizes(w, constants.DefaultMaxPayloadSize, request.Username) {
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
		sendResponse(w, models.ResponseSuccess, "", models.LoginResponse{
			Token: token,
		})
	} else {
		sendResponse(w, models.ResponseError, "Error logging in", nil, 403)
	}
}

//Register register handler
//-> /user/create
func Register(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
	if !handlerData.config.Server.AllowRegistration {
		sendResponse(w, models.ResponseError, "Server doesn't accept registrations", nil, 403)
		return
	}

	var request models.CredentialRequest

	if !parseUserInput(handlerData.config, w, r, &request) {
		return
	}

	if isStructInvalid(request) || len(request.Password) != 128 || len(request.Username) > 30 {
		sendError("input missing", w, models.WrongInputFormatError, 422)
		return
	}

	exists, err := models.UserExists(db, request.Username)
	if err != nil {
		sendServerError(w)
		return
	}

	if exists {
		sendResponse(w, models.ResponseError, "User exists", nil)
		return
	}

	err = models.InsertUser(db, request.Username, gaw.SHA512(request.Password+request.Username), gaw.GetIPFromHTTPrequest(r))
	if err != nil {
		sendServerError(w)
		return
	}

	sendResponse(w, models.ResponseSuccess, "", nil)
}
