package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type handlerData struct {
	config             *models.ConfigStruct
	ownIP              *string
	user               *models.User
	subscriberCallback models.SubscriberNotifyCallback
}

//Route for REST
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc RouteFunction
	HandlerType handlerType
}

type handlerType uint8

const (
	defaultHandlerType handlerType = iota
	sessionHandlerType
)

//Routes all REST routes
type Routes []Route

//RouteFunction function for handling a route
type RouteFunction func(*dbhelper.DBhelper, handlerData, http.ResponseWriter, *http.Request)

//Routes
var routes = Routes{
	//User
	Route{"login", "POST", "/user/login", Login, defaultHandlerType},
	Route{"register", "POST", "/user/create", Register, defaultHandlerType},

	//Sources
	Route{"create source", "POST", "/source/create", CreateSource, sessionHandlerType},
	Route{"update source", "POST", "/source/update/{action}", UpdateSource, sessionHandlerType},
	Route{"listSources", "POST", "/sources", ListSources, sessionHandlerType},

	//Subscriptions
	Route{"subscribe", "POST", "/sub/add", Subscribe, defaultHandlerType},
	Route{"unsubscribe", "POST", "/sub/remove", Unsubscribe, defaultHandlerType},
	Route{"update callback", "POST", "/sub/updateCallback", UpdateCallbackURL, defaultHandlerType},

	//Webhook
	Route{"Post webhook", "POST", "/webhook/post/{sourceID}/{secret}", WebhookHandler, defaultHandlerType},
	Route{"GET webhook", "GET", "/webhook/get/{sourceID}/{secret}", WebhookHandler, defaultHandlerType},
}

//NewRouter create new router
func NewRouter(db *dbhelper.DBhelper, config *models.ConfigStruct, ownIP *string, callback models.SubscriberNotifyCallback) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(RouteHandler(db, route.HandlerType, &handlerData{
				config:             config,
				subscriberCallback: callback,
				ownIP:              ownIP,
			}, route.HandlerFunc, route.Name))
	}
	return router
}

//RouteHandler logs stuff
func RouteHandler(db *dbhelper.DBhelper, handlerType handlerType, handlerData *handlerData, inner RouteFunction, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("[%s] %s\n", r.Method, name)

		start := time.Now()

		if validateHeader(handlerData.config, w, r) {
			return
		}

		//Validate handlerType
		if !handlerType.validate(db, handlerData, r, w) {
			return
		}

		//Process request
		inner(db, *handlerData, w, r)

		//Print duration of processing
		printProcessingDuration(start)
	})
}

//Returns false on error
func (handlerType handlerType) validate(db *dbhelper.DBhelper, handlerData *handlerData, r *http.Request, w http.ResponseWriter) bool {
	switch handlerType {
	case sessionHandlerType:
		{
			//Get auth header
			authHeader, has := r.Header["Authorization"]
			//Validate bearer token
			if !has || len(authHeader) == 0 || !strings.HasPrefix(authHeader[0], "Bearer") || len(tokenFromBearerHeader(authHeader[0])) != 64 {
				sendResponse(w, models.ResponseError, models.InvalidTokenError, nil, http.StatusUnauthorized)
				return false
			}

			user, _ := models.GetUserBySession(db, tokenFromBearerHeader(authHeader[0]))
			if user == nil {
				sendResponse(w, models.ResponseError, models.InvalidTokenError, nil, http.StatusUnauthorized)
				return false
			}

			//Update IP
			go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

			//Set user
			handlerData.user = user
		}
	}

	return true
}

//Prints the duration of handling the function
func printProcessingDuration(startTime time.Time) {
	dur := time.Since(startTime)

	if dur < 1500*time.Millisecond {
		log.Debugf("Duration: %s\n", dur.String())
	} else if dur > 1500*time.Millisecond {
		log.Warningf("Duration: %s\n", dur.String())
	}
}

//Return true on error
func validateHeader(config *models.ConfigStruct, w http.ResponseWriter, r *http.Request) bool {
	headerSize := getHeaderSize(r.Header)

	//Send error if header are too big. MaxHeaderLength is stored in b
	if headerSize > uint32(config.Webserver.MaxHeaderLength) {
		//Send error response
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		fmt.Fprint(w, "413 request too large")

		log.Warnf("Got request with %db headers. Maximum allowed are %db\n", headerSize, config.Webserver.MaxHeaderLength)
		return true
	}

	return false
}
