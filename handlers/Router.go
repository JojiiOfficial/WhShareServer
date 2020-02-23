package handlers

import (
	"fmt"
	"net/http"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type handlerData struct {
	config             *models.ConfigStruct
	ownIP              *string
	subscriberCallback models.SubscriberNotifyCallback
}

//Route for REST
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc RouteFunction
}

//Routes all REST routes
type Routes []Route

//RouteFunction function for handling a route
type RouteFunction func(*dbhelper.DBhelper, handlerData, http.ResponseWriter, *http.Request)

//Routes
var routes = Routes{
	//User
	Route{"login", "POST", "/user/login", Login},
	Route{"register", "POST", "/user/create", Register},

	//Sources
	Route{"create source", "POST", "/source/create", CreateSource},
	Route{"update source", "POST", "/source/update/{action}", UpdateSource},
	Route{"listSources", "POST", "/sources", ListSources},

	//Subscriptions
	Route{"subscribe", "POST", "/sub/add", Subscribe},
	Route{"unsubscribe", "POST", "/sub/remove", Unsubscribe},
	Route{"update callback", "POST", "/sub/updateCallback", UpdateCallbackURL},

	//Webhook
	Route{"Post webhook", "POST", "/webhook/post/{sourceID}/{secret}", WebhookHandler},
	Route{"GET webhook", "GET", "/webhook/get/{sourceID}/{secret}", WebhookHandler},
}

//NewRouter create new router
func NewRouter(db *dbhelper.DBhelper, config *models.ConfigStruct, ownIP *string, callback models.SubscriberNotifyCallback) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(RouteHandler(db, &handlerData{
				config:             config,
				subscriberCallback: callback,
				ownIP:              ownIP,
			}, route.HandlerFunc, route.Name))
	}
	return router
}

//RouteHandler logs stuff
func RouteHandler(db *dbhelper.DBhelper, handlerData *handlerData, inner RouteFunction, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("[%s] %s\n", r.Method, name)

		start := time.Now()

		if validateHeader(handlerData.config, w, r) {
			return
		}

		//Process request
		inner(db, *handlerData, w, r)

		//Print duration of processing
		printProcessingDuration(start)
	})
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
