package handlers

import (
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/constants"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/gorilla/mux"
)

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
		sendResponse(w, models.ResponseError, "404 Not found", nil, 404)
		return
	}

	if source.Secret == secret {
		c := make(chan bool, 1)
		log.Infof("New webhook: %s\n", source.Name)
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
			if isHeaderBlocklistetd(req.Header, &handlerData.config.Server.WebhookBlacklist.HeaderValues) {
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
			payload, err := ioutil.ReadAll(io.LimitReader(req.Body, handlerData.config.Webserver.MaxPayloadBodyLength))
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
			payload, err = gaw.JSONRemoveItems(payload, handlerData.config.Server.WebhookBlacklist.JSONObjects[constants.ModeToString[source.Mode]], false)
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

			handlerData.subscriberCallback.OnWebhookReceive(webhook, source)
		})(r)

		if <-c {
			sendResponse(w, models.ResponseSuccess, "Success", nil)
		} else {
			sendResponse(w, models.ResponseError, msg, nil, 500)
		}

	} else {
		log.Warn("invalid secret for source", sourceID)
	}
}
