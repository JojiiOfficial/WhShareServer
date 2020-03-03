package handlers

import (
	"fmt"
	"net/http"
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
	Method      HTTPMethod
	Pattern     string
	HandlerFunc RouteFunction
	HandlerType requestType
}

//HTTPMethod http method. GET, POST, DELETE, HEADER, etc...
type HTTPMethod string

//HTTP methods
const (
	GetMethod    HTTPMethod = "GET"
	POSTMethod   HTTPMethod = "POST"
	DeleteMethod HTTPMethod = "DELETE"
)

type requestType uint8

const (
	defaultRequest requestType = iota
	sessionRequest
	optionalTokenRequest
)

//Routes all REST routes
type Routes []Route

//RouteFunction function for handling a route
type RouteFunction func(*dbhelper.DBhelper, handlerData, http.ResponseWriter, *http.Request)

//Routes
var (
	routes = Routes{
		//User
		Route{
			Name:        "login",
			Pattern:     "/user/login",
			Method:      POSTMethod,
			HandlerFunc: Login,
			HandlerType: defaultRequest,
		},
		Route{
			Name:        "register",
			Pattern:     "/user/create",
			Method:      POSTMethod,
			HandlerFunc: Register,
			HandlerType: defaultRequest,
		},

		//Sources
		Route{
			Name:        "create source",
			Pattern:     "/source/create",
			Method:      POSTMethod,
			HandlerFunc: CreateSource,
			HandlerType: sessionRequest,
		},
		Route{
			Name:        "update source",
			Pattern:     "/source/update/{action}",
			Method:      POSTMethod,
			HandlerFunc: UpdateSource,
			HandlerType: sessionRequest,
		},
		Route{
			Name:        "list sources",
			Pattern:     "/sources",
			Method:      POSTMethod,
			HandlerFunc: ListSources,
			HandlerType: sessionRequest,
		},

		//Subscriptions
		Route{
			Name:        "subscribe",
			Pattern:     "/sub/add",
			Method:      POSTMethod,
			HandlerFunc: Subscribe,
			HandlerType: optionalTokenRequest,
		},
		Route{
			Name:        "unsubscribe",
			Pattern:     "/sub/remove",
			Method:      POSTMethod,
			HandlerFunc: Unsubscribe,
			HandlerType: optionalTokenRequest,
		},
		Route{
			Name:        "update callback",
			Pattern:     "/sub/updateCallback",
			Method:      POSTMethod,
			HandlerFunc: UpdateCallbackURL,
			HandlerType: optionalTokenRequest,
		},

		//Webhooks
		Route{"Post webhook", "POST", "/webhook/post/{sourceID}/{secret}", WebhookHandler, defaultRequest},
		Route{"GET webhook", "GET", "/webhook/get/{sourceID}/{secret}", WebhookHandler, defaultRequest},
		//With params
		Route{"POST webhook params", "POST", "/webhook/post/{sourceID}/{secret}/{params}", WebhookHandler, defaultRequest},
		Route{"GET webhook params", "GET", "/webhook/get/{sourceID}/{secret}/{params}", WebhookHandler, defaultRequest},
	}
)

//NewRouter create new router
func NewRouter(db *dbhelper.DBhelper, config *models.ConfigStruct, ownIP *string, callback models.SubscriberNotifyCallback) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(string(route.Method)).
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
func RouteHandler(db *dbhelper.DBhelper, requestType requestType, handlerData *handlerData, inner RouteFunction, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("[%s] %s\n", r.Method, name)

		start := time.Now()

		if validateHeader(handlerData.config, w, r) {
			return
		}

		//Validate request by requestType
		if !requestType.validate(db, handlerData, r, w) {
			return
		}

		//Process request
		inner(db, *handlerData, w, r)

		//Print duration of processing
		printProcessingDuration(start)
	})
}

//Return false on error
func (requestType requestType) validate(db *dbhelper.DBhelper, handlerData *handlerData, r *http.Request, w http.ResponseWriter) bool {
	switch requestType {
	case sessionRequest:
		{
			authHandler := NewAuthHandler(r, db)
			user, err := authHandler.GetUserFromBearer()

			//validate user
			if err != nil {
				sendResponse(w, models.ResponseError, models.InvalidTokenError, nil, http.StatusUnauthorized)
				return false
			}

			//Return error if user is invalid
			if !user.IsValid {
				sendResponse(w, models.ResponseError, models.UserIsInvalidErr, nil, http.StatusUnauthorized)
				return false
			}

			//Update IP
			go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))

			//Set user
			handlerData.user = user
		}
	case optionalTokenRequest:
		{
			authHandler := NewAuthHandler(r, db)
			user, err := authHandler.GetUserFromBearer()

			//Return error if token is provided but invalid
			if authHandler.IsInvalid(err) {
				sendResponse(w, models.ResponseError, models.InvalidTokenError, nil, http.StatusUnauthorized)
				return false
			}

			//Return error if user is invalid
			if user != nil {
				if !user.IsValid {
					sendResponse(w, models.ResponseError, models.UserIsInvalidErr, nil, http.StatusUnauthorized)
					return false
				}

				//Update users IP address
				go user.UpdateIP(db, gaw.GetIPFromHTTPrequest(r))
			}

			//just set the user. If nil, no user was provided
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
