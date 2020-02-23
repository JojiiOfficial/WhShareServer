package services

import (
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	log "github.com/sirupsen/logrus"
)

//ResetUsageService the service to reset users traffic/hookCalls
type ResetUsageService struct {
	db *dbhelper.DBhelper
}

//NewResetUsageService create new ResetUsageService
func NewResetUsageService(db *dbhelper.DBhelper) *ResetUsageService {
	return &ResetUsageService{
		db: db,
	}
}

//Tick runs the action of the service
func (service ResetUsageService) Tick() <-chan bool {
	c := make(chan bool)

	go (func() {
		start := time.Now()
		n, err := service.reset()

		if err == nil && n > 0 {
			dur := time.Now().Sub(start).String()
			log.Debugf("Resource usage resetting took %s\n", dur)
			log.Infof("Reset resource usage for %d user(s)", n)
		}

		c <- true
	})()

	return c
}

func (service ResetUsageService) reset() (int64, error) {
	rs, err := service.db.Execf(
		"UPDATE %s SET resetIndex=TIMESTAMPDIFF(MONTH, createdAt, now()), traffic=0, hookCalls=0 WHERE TIMESTAMPDIFF(MONTH, createdAt, now()) > resetIndex",
		[]string{models.TableUser},
	)

	if err != nil {
		return 0, err
	}
	return rs.RowsAffected()
}
