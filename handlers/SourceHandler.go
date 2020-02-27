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
func CreateSource(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
	var request models.SourceAddRequest

	if !parseUserInput(handlerData.config, w, r, &request) {
		return
	}

	if checkInput(w, request, request.Name) {
		return
	}

	//Check if user is allowed to create sources
	if !handlerData.user.CanCreateSource(request.Private) {
		sendResponse(w, models.ResponseError, "You are not allowed to have this kind of source", nil, http.StatusForbidden)
		return
	}

	//Check if user source limit is reached
	isLimitReached, err := handlerData.user.IsSourceLimitReached(db, request.Private)
	if err != nil {
		//TODO send user not found
		sendServerError(w)
		return
	}

	if isLimitReached {
		sendResponse(w, models.ResponseError, "Limit for this kind of source exceeded", nil, http.StatusForbidden)
		return
	}

	//Check if user already has a source with this name
	nameExitst, err := handlerData.user.HasSourceWithName(db, request.Name)
	if err != nil {
		sendServerError(w)
		return
	}

	if nameExitst {
		sendResponse(w, models.ResponseError, models.MultipleSourceNameErr, nil)
		return
	}

	source := &models.Source{
		Creator:     *handlerData.user,
		CreatorID:   handlerData.user.Pkid,
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
func ListSources(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
	var request models.SourceRequest

	if !parseUserInput(handlerData.config, w, r, &request) {
		return
	}

	if request.SourceID == "-" {
		request.SourceID = ""
	}

	var response models.ListSourcesResponse

	if len(request.SourceID) == 0 {
		//No sourceID provided
		sources, err := models.GetSourcesForUser(db, handlerData.user.Pkid)
		if err != nil {
			sendServerError(w)
			return
		}

		response = models.ListSourcesResponse{
			Sources: sources,
		}

	} else {
		//SourceID provided
		source, err := models.GetSourceFromSourceID(db, request.SourceID)
		if err != nil {
			sendServerError(w)
			return
		}

		if handlerData.user.Pkid != source.CreatorID {
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
func UpdateSource(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
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
	if !parseUserInput(handlerData.config, w, r, &request) {
		return
	}

	if checkInput(w, request, request.Content) {
		return
	}

	source, err := models.GetSourceFromSourceID(db, request.SourceID)
	if err != nil {
		if err.Error() == dbhelper.ErrNoRowsInResultSet {
			//Source just not found if no rows in result set
			sendResponse(w, models.ResponseError, models.NotFoundError, nil, http.StatusNotFound)
			return
		}
		//Real db error
		sendServerError(w)
		return
	}

	if source.CreatorID != handlerData.user.Pkid {
		sendError("user not allowed", w, models.ActionNotAllowed, http.StatusForbidden)
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
			has, err := handlerData.user.HasSourceWithName(db, request.Content)
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
