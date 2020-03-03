package handlers

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/constants"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/gorilla/mux"
)

type webhookResp struct {
	StatusCode int
	Message    string
}

//WebhookHandler handler for incoming webhooks
//-> /post/webhook
func WebhookHandler(db *dbhelper.DBhelper, handlerData handlerData, w http.ResponseWriter, r *http.Request) {
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
		sendResponse(w, models.ResponseError, "404 Not found", nil, http.StatusNotFound)
		return
	}

	if source.Secret == secret {
		c := make(chan webhookResp, 1)
		log.Infof("New webhook: %s\n", source.Name)

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
			if isHeaderBlocklistetd(req.Header, &handlerData.config.Server.WebhookBlacklist.HeaderValues) {
				log.Warnf("Blocked webhook '%s' because of header-blacklist\n", source.SourceID)
				c <- webhookResp{StatusCode: http.StatusAccepted, Message: "Content won't forwarded"}
				return
			}

			//Await getting user
			user := <-userChan

			//Validate user
			if user == nil {
				c <- webhookResp{
					StatusCode: http.StatusBadRequest,
					Message:    "User not found",
				}
				return
			}

			//return error if user not allowed to send hooks
			if !user.CanShareWebhooks() {
				c <- webhookResp{StatusCode: http.StatusMethodNotAllowed, Message: "not allowed to send webhooks"}
				return
			}

			//Read payload from webhook
			payload, err := ioutil.ReadAll(io.LimitReader(req.Body, handlerData.config.Webserver.MaxPayloadBodyLength))
			if err != nil {
				LogError(err)
				c <- webhookResp{StatusCode: http.StatusInternalServerError, Message: "error reading payload"}
				return
			}
			req.Body.Close()

			headers := headerToString(req.Header)

			//Calculate traffic of request
			reqTraffic := uint32(len(payload)) + uint32(len(headers))

			//Check if user limit exceeded
			if (user.Role.MaxTraffic != -1 && uint32(user.Role.MaxTraffic*1024) <= (user.Traffic+reqTraffic)) ||
				(user.Role.MaxHookCalls != -1 && user.Role.MaxHookCalls < int(user.HookCalls+1)) {
				c <- webhookResp{StatusCode: http.StatusForbidden, Message: "traffic/hookCall limit exceeded"}
				return
			}

			//Delete in config specified json objects
			payload, err = gaw.JSONRemoveItems(payload, handlerData.config.Server.WebhookBlacklist.JSONObjects[constants.ModeToString[source.Mode]], false)
			if err != nil {
				LogError(err, log.Fields{"msg": "Error filtering JSON!"})
				c <- webhookResp{StatusCode: http.StatusInternalServerError, Message: "server error"}
				return
			}

			//Send success
			c <- webhookResp{
				StatusCode: http.StatusOK,
				Message:    "Success",
			}

			//Update traffic and hookCallCount if not both unlimited
			if !user.HasUnlimitedHookCalls() {
				user.AddHookCall(db, reqTraffic)
			}

			//append params
			if p, has := vars["params"]; has {
				payload, err = appendJSONKeys(payload, strings.Split(p, "&")...)
			}

			webhook := &models.Webhook{
				SourceID: source.PkID,
				Headers:  headers,
				Payload:  string(payload),
			}
			webhook.Insert(db)

			handlerData.subscriberCallback.OnWebhookReceive(webhook, source)
		})(r)

		res := <-c
		http.Error(w, res.Message, res.StatusCode)

	} else {
		log.Warn("invalid secret for source", sourceID)
	}
}
