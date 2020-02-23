package services

import (
	//log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	log "github.com/sirupsen/logrus"
)

//CleanupService cleans up stuff
type CleanupService struct {
	db *dbhelper.DBhelper
}

//NewCleanupService create a new cleanup service
func NewCleanupService(db *dbhelper.DBhelper) *CleanupService {
	return &CleanupService{
		db: db,
	}
}

//Tick runs the action of the service
func (service CleanupService) Tick() <-chan bool {
	c := make(chan bool)

	go (func() {
		service.clean()
		c <- true
	})()

	return c
}

func (service CleanupService) clean() {
	_, err := service.db.Execf("DELETE FROM %s WHERE (%s.received < (SELECT MIN(lastTrigger) FROM %s WHERE %s.source = %s.sourceID) AND DATE_ADD(received, INTERVAL 1 day) <= now()) OR DATE_ADD(received, INTERVAL 2 day) <= now()", []string{models.TableWebhooks, models.TableWebhooks, models.TableSubscriptions, models.TableSubscriptions, models.TableWebhooks})
	if err != nil {
		LogError(err)
	}
	log.Info("Webhook cleanup done")
}