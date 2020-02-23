package handlers

import (
	"net/http"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/gorilla/mux"
)

//CreateSource creates a source
//-> /source/create
func CreateSource(db *dbhelper.DBhelper, config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) {
	var request models.SourceAddRequest
	if !parseUserInput(config, w, r, &request) {
		return
	}

	if checkInput(w, request, request.Token, request.Name) {
		return
	}

	user, err := models.GetUserBySession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, models.InvalidTokenError, 403)
		return
	}

	go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

	//Check if user is allowed to create sources
	if request.Private && user.Role.MaxPrivSources == 0 {
		sendResponse(w, models.ResponseError, "You are not allowed to have private sources", nil, 403)
		return
	} else if !request.Private && user.Role.MaxPubSources == 0 {
		sendResponse(w, models.ResponseError, "You are not allowed to have public sources", nil, 403)
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
			sendResponse(w, models.ResponseError, "Limit for private sources exceeded", nil, 403)
			return
		}
	} else if !request.Private && user.Role.MaxPubSources != -1 {
		if scount >= uint(user.Role.MaxPubSources) {
			sendResponse(w, models.ResponseError, "Limit for public sources exceeded", nil, 403)
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
		sendResponse(w, models.ResponseError, models.MultipleSourceNameErr, nil)
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

	sendResponse(w, models.ResponseSuccess, "", models.SourceAddResponse{
		Secret:   source.Secret,
		SourceID: source.SourceID,
	})
}

//ListSources lists sources
//-> /sources
func ListSources(db *dbhelper.DBhelper, config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) {
	var request models.SourceRequest

	if !parseUserInput(config, w, r, &request) {
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
		sendError("Invalid token", w, models.InvalidTokenError, 403)
		return
	}

	go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

	var response models.ListSourcesResponse
	if len(request.SourceID) == 0 {
		sources, err := models.GetSourcesForUser(db, user.Pkid)
		if err != nil {
			sendServerError(w)
			return
		}

		response = models.ListSourcesResponse{
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

		response = models.ListSourcesResponse{
			Sources: []models.Source{*source},
		}
	}

	sendResponse(w, models.ResponseSuccess, "", response)
}

//UpdateSource updates a source
//-> /source/update/{action}
func UpdateSource(db *dbhelper.DBhelper, config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	action := vars["action"]
	actions := []string{
		"delete",
		"changedescr",
		"rename",
		"toggleAccess",
	}

	if !gaw.IsInStringArray(action, actions) {
		sendError("not available", w, models.WrongInputFormatError, 501)
		return
	}

	var request models.SourceRequest
	if !parseUserInput(config, w, r, &request) {
		return
	}

	if checkInput(w, request, request.Token, request.Content) {
		return
	}

	user, err := models.GetUserBySession(db, request.Token)
	if err != nil {
		sendError("Invalid token", w, models.InvalidTokenError, 403)
		return
	}

	go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		if err.Error() == dbhelper.ErrNoRowsInResultSet {
			//Source just not found if no rows in result set
			sendResponse(w, models.ResponseError, models.NotFoundError, nil, 404)
			return
		}
		//Real db error
		sendServerError(w)
		return
	}

	if source.CreatorID != user.Pkid {
		sendError("user not allowed", w, models.ActionNotAllowed, 403)
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
				sendResponse(w, models.ResponseError, models.MultipleSourceNameErr, nil)
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
		sendResponse(w, models.ResponseSuccess, message, nil)
	}
}
