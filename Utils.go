package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	gaw "github.com/JojiiOfficial/GoAw"
)

func getDataPath() string {
	path := path.Join(gaw.GetHome(), DataDir)
	s, err := os.Stat(path)
	if err != nil {
		err = os.Mkdir(path, 0770)
		if err != nil {
			log.Fatalln(err.Error())
		}
	} else if s != nil && !s.IsDir() {
		log.Fatalln("DataPath-name already taken by a file!")
	}
	return path
}

func headerToString(headers http.Header) string {
	var sheaders string
	for k, v := range headers {
		sheaders += k + "=" + strings.Join(v, ";") + "\r\n"
	}
	return sheaders
}

func setHeadersFromStr(headers string, header *http.Header) {
	headersrn := strings.Split(headers, "\r\n")
	for _, v := range headersrn {
		if !strings.Contains(v, "=") {
			continue
		}
		kp := strings.Split(v, "=")
		key := kp[0]

		(*header).Set(key, kp[1])
	}
}

//Returns the size in bytes of the header
func getHeaderSize(headers http.Header) uint32 {
	var size uint32
	for k, v := range headers {
		size += uint32(len(k))
		for _, val := range v {
			size += uint32(len(val))
		}
	}
	return size
}

func isHeaderBlocklistetd(headers http.Header, blocklist *map[string][]string) bool {
	start := time.Now()

	for k, headerValues := range headers {
		blocklistValues, ok := (*blocklist)[strings.ToLower(k)]
		if ok {
			for _, headerValue := range headerValues {
				for _, blocklistValue := range blocklistValues {
					if strings.ToLower(blocklistValue) == strings.ToLower(headerValue) {
						return true
					}
				}
			}
		}
	}

	dur := time.Now().Sub(start)
	//Print only if 'critical'
	if dur >= 1*time.Second {
		log.Warnf("Header checking took %s\n", dur.String())
	}

	return false
}

//LogError returns true on error
func LogError(err error, context ...map[string]interface{}) bool {
	if err == nil {
		return false
	}

	if len(context) > 0 {
		log.WithFields(context[0]).Error(err.Error())
	} else {
		log.Error(err.Error())
	}
	return true
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
	return net.ParseIP(inp).To4() != nil
}

//Return true if valid
func isValidCallback(inp string, allowBogon bool, addIPs ...string) (bool, error) {
	inp = strings.TrimSpace(inp)
	if !isValidHTTPURL(inp) {
		return false, nil
	}

	u, err := url.Parse(inp)
	if err != nil {
		return false, err
	}

	host := u.Hostname()

	//If inp is an IP
	if isIPv4(host) {
		//Check for bogon
		isReserved, err := gaw.IsReserved(host)
		if err != nil || (isReserved && !allowBogon) {
			return false, err
		}

		//Check additional IPs
		for _, addIP := range addIPs {
			addIP = strings.TrimSpace(addIP)
			if len(addIP) == 0 {
				continue
			}
			if strings.TrimSpace(addIP) == host {
				return false, nil
			}
		}

		//Otherwise return valid
		return true, nil
	}

	//If host, do DNS lookup
	ips, err := net.LookupHost(host)
	if err != nil {
		//If server can't lookup the host, then the host is not valid
		return false, nil
	}

	//Loop DNS IPs and check if is reserved
	for _, ipp := range ips {
		isReserved, err := gaw.IsIPReserved(strings.TrimSpace(ipp))
		if err != nil || (isReserved && !allowBogon) {
			return false, err
		}
	}

	//Loop DNS IPs and compare with add IPs
	for _, ipp := range ips {
		for _, addIP := range addIPs {
			addIP = strings.TrimSpace(addIP)
			if len(addIP) == 0 {
				continue
			}
			if addIP == strings.TrimSpace(ipp) {
				return false, nil
			}
		}
	}

	return true, nil
}

//AllowedSchemes schemes that are allowed in urls
var AllowedSchemes = []string{"http", "https"}

func isValidHTTPURL(inp string) bool {
	//check for valid URL
	u, err := url.Parse(inp)
	if err != nil {
		return false
	}

	return gaw.IsInStringArray(u.Scheme, AllowedSchemes)
}
