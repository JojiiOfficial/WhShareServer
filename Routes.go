package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

//Route for REST
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

//Routes all REST routes
type Routes []Route

//NewRouter create new router
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	return router
}

var routes = Routes{
	//User
	Route{
		"login",
		"POST",
		"/user/login",
		login,
	},
	Route{
		"register",
		"POST",
		"/user/create",
		register,
	},

	//Sources
	Route{
		"create source",
		"POST",
		"/source/create",
		createSource,
	},
	Route{
		"update source",
		"POST",
		"/source/update/{action}",
		updateSource,
	},
	Route{
		"listSources",
		"POST",
		"/sources",
		listSources,
	},

	//Subscriptions
	Route{
		"subscribe",
		"POST",
		"/sub/add",
		subscribe,
	},
	Route{
		"unsubscribe",
		"POST",
		"/sub/remove",
		unsubscribe,
	},

	//Webhook
	//Ending without /
	Route{
		"Post webhook",
		"POST",
		"/webhook/post/{sourceID}/{secret}",
		webhookHandler,
	},
	Route{
		"GET webhook",
		"GET",
		"/webhook/get/{sourceID}/{secret}",
		webhookHandler,
	},

	//Ending with /
	Route{
		"Post webhook",
		"POST",
		"/webhook/post/{sourceID}/{secret}/",
		webhookHandler,
	},
	Route{
		"GET webhook",
		"GET",
		"/webhook/get/{sourceID}/{secret}/",
		webhookHandler,
	},
}
