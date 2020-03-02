package services

import (
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	log "github.com/sirupsen/logrus"
)

//CleanupService cleans up stuff
type CleanupService struct {
	db     *dbhelper.DBhelper
	config *models.ConfigStruct
}

//NewCleanupService create a new cleanup service
func NewCleanupService(db *dbhelper.DBhelper, config *models.ConfigStruct) *CleanupService {
	return &CleanupService{
		db:     db,
		config: config,
	}
}

//Tick runs the action of the service
func (service CleanupService) Tick() <-chan error {
	c := make(chan error)

	go (func() {
		err := service.clean()
		log.Info("Webhook cleanup done")
		c <- err
	})()

	return c
}

func (service CleanupService) clean() error {
	//Magic query. Cleans up old webhooks
	_, err := service.db.Execf("DELETE FROM %s WHERE (%s.received < (SELECT MIN(lastTrigger) FROM %s WHERE %s.source = %s.sourceID) AND DATE_ADD(received, INTERVAL 1 day) <= now()) OR DATE_ADD(received, INTERVAL 2 day) <= now()", []string{models.TableWebhooks, models.TableWebhooks, models.TableSubscriptions, models.TableSubscriptions, models.TableWebhooks})
	if err != nil {
		return err
	}

	//Delete old loginsessions
	if service.config.Server.CleanSessionsAfter.Seconds() > 0 {
		minTime := time.Now().Unix() - int64(service.config.Server.CleanSessionsAfter.Seconds())
		_, err = service.db.Execf("DELETE FROM %s WHERE lastAccessed < from_unixtime(?)", []string{models.TableLoginSession}, minTime)
	} else {
		log.Debug("Not cleaning sessions")
	}

	return err
}
