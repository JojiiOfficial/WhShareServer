package services

import (
	"io/ioutil"
	"net"
	"net/http"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	log "github.com/sirupsen/logrus"
)

//IPRefreshService the service to keep the current IP validated
type IPRefreshService struct {
	db *dbhelper.DBhelper
	IP string
}

//NewIPRefreshService create a new IPRefreshService
func NewIPRefreshService(db *dbhelper.DBhelper) *IPRefreshService {
	return &IPRefreshService{
		db: db,
	}
}

//Init inits the IPRefreshService. Return true on success
func (service *IPRefreshService) Init() bool {
	service.IP = getOwnIP()
	return isIPv4(service.IP)
}

//Tick runs the action of the service
func (service IPRefreshService) Tick() <-chan bool {
	c := make(chan bool)

	go (func() {
		service.refresh()
		c <- true
	})()

	return c
}

func (service *IPRefreshService) refresh() {
	getIP := getOwnIP()

	if getIP != service.IP && isIPv4(getIP) {
		log.Infof("Server got new IP address %s\n", getIP)
		service.IP = getIP
	}
}

func getOwnIP() string {
	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	cnt, _ := ioutil.ReadAll(resp.Body)
	return string(cnt)
}

func isIPv4(inp string) bool {
	if len(inp) < 7 || len(inp) > 15 {
		return false
	}
	return net.ParseIP(inp).To4() != nil
}
