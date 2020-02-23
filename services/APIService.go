package services

import (
	"net/http"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
)

//APIService the service handling the API
type APIService struct {
	router *mux.Router
	db     *dbhelper.DBhelper
	config *models.ConfigStruct
}

//NewAPIService create new API service
func NewAPIService(db *dbhelper.DBhelper, config *models.ConfigStruct, router *mux.Router) *APIService {
	return &APIService{
		db:     db,
		config: config,
		router: router,
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
