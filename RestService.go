package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

//TODO move to services package

//Route for REST
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

//Routes all REST routes
type Routes []Route

//Routes
var routes = Routes{
	//User
	Route{"login", "POST", "/user/login", login},
	Route{"register", "POST", "/user/create", register},

	//Sources
	Route{"create source", "POST", "/source/create", createSource},
	Route{"update source", "POST", "/source/update/{action}", updateSource},
	Route{"listSources", "POST", "/sources", listSources},

	//Subscriptions
	Route{"subscribe", "POST", "/sub/add", subscribe},
	Route{"unsubscribe", "POST", "/sub/remove", unsubscribe},
	Route{"update callback", "POST", "/sub/updateCallback", updateCallbackURL},

	//Webhook
	//Ending without /
	Route{"Post webhook", "POST", "/webhook/post/{sourceID}/{secret}", webhookHandler},
	Route{"GET webhook", "GET", "/webhook/get/{sourceID}/{secret}", webhookHandler},

	//Ending with /
	Route{"Post webhook", "POST", "/webhook/post/{sourceID}/{secret}/", webhookHandler},
	Route{"GET webhook", "GET", "/webhook/get/{sourceID}/{secret}/", webhookHandler},
}

//NewRouter create new router
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(RouteHandler(route.HandlerFunc, route.Name))
	}
	return router
}

//RouteHandler logs stuff
func RouteHandler(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info(r.Method + " " + r.RequestURI + " " + name)

		headerSize := getHeaderSize(r.Header)
		//Send error if header are too big. MaxHeaderLength is stored in b
		if headerSize > uint32(config.Webserver.MaxHeaderLength) {
			//Send error response
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			fmt.Fprint(w, "413 request too large")

			//Log
			log.Warnf("Got request with %db headers. Maximum allowed are %db\n", headerSize, config.Webserver.MaxHeaderLength)
			return
		}

		start := time.Now()
		inner.ServeHTTP(w, r)
		dur := time.Since(start)
		if dur < 1500*time.Millisecond {
			log.Debugf("Duration: %s\n", dur.String())
		} else if dur > 1500*time.Millisecond {
			log.Warningf("Duration: %s\n", dur.String())
		}
	})
}
