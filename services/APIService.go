package services

import (
	"fmt"
	"net/http"

	"time"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/handlers"
	"github.com/JojiiOfficial/WhShareServer/models"
)

//APIService the service handling the API
type APIService struct {
	router *mux.Router
	db     *dbhelper.DBhelper
	config *models.ConfigStruct
}

//NewAPIService create new API service
func NewAPIService(db *dbhelper.DBhelper, config *models.ConfigStruct) *APIService {
	return &APIService{
		db:     db,
		config: config,
		router: NewRouter(db, config),
	}
}

//Start the API service
func (service *APIService) Start() {
	//Start HTTPS if enabled
	if service.config.Webserver.HTTPS.Enabled {
		log.Infof("Server started TLS on port (%s)\n", service.config.Webserver.HTTPS.ListenAddress)
		go (func() {
			log.Fatal(http.ListenAndServeTLS(service.config.Webserver.HTTPS.ListenAddress, service.config.Webserver.HTTPS.CertFile, service.config.Webserver.HTTPS.KeyFile, service.router))
		})()
	}

	//Start HTTP if enabled
	if service.config.Webserver.HTTP.Enabled {
		log.Infof("Server started HTTP on port (%s)\n", service.config.Webserver.HTTP.ListenAddress)
		go (func() {
			log.Fatal(http.ListenAndServe(service.config.Webserver.HTTP.ListenAddress, service.router))
		})()
	}
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
type RouteFunction func(*dbhelper.DBhelper, *models.ConfigStruct, http.ResponseWriter, *http.Request)

//Routes
var routes = Routes{
	//User
	Route{"login", "POST", "/user/login", handlers.Login},
	Route{"register", "POST", "/user/create", handlers.Register},

	//Sources
	Route{"create source", "POST", "/source/create", handlers.CreateSource},
	Route{"update source", "POST", "/source/update/{action}", handlers.UpdateSource},
	Route{"listSources", "POST", "/sources", handlers.ListSources},

	//Subscriptions
	Route{"subscribe", "POST", "/sub/add", handlers.Subscribe},
	Route{"unsubscribe", "POST", "/sub/remove", handlers.Unsubscribe},
	Route{"update callback", "POST", "/sub/updateCallback", handlers.UpdateCallbackURL},

	//Webhook
	Route{"Post webhook", "POST", "/webhook/post/{sourceID}/{secret}", handlers.WebhookHandler},
	Route{"GET webhook", "GET", "/webhook/get/{sourceID}/{secret}", handlers.WebhookHandler},
}

//NewRouter create new router
func NewRouter(db *dbhelper.DBhelper, config *models.ConfigStruct) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(RouteHandler(db, config, route.HandlerFunc, route.Name))
	}
	return router
}

//RouteHandler logs stuff
func RouteHandler(db *dbhelper.DBhelper, config *models.ConfigStruct, inner RouteFunction, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("[%s] %s\n", r.Method, name)

		start := time.Now()

		if validateHeader(config, w, r) {
			return
		}

		//Process request
		inner(db, config, w, r)

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
