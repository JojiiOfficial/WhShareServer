package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	log "github.com/sirupsen/logrus"

	"github.com/JojiiOfficial/WhShareServer/constants"
	"github.com/JojiiOfficial/WhShareServer/models"
)

func sendResponse(w http.ResponseWriter, status models.ResponseStatus, message string, payload interface{}, params ...int) {
	statusCode := http.StatusOK
	s := "0"
	if status == 1 {
		s = "1"
	}

	w.Header().Set(models.HeaderStatus, s)
	w.Header().Set(models.HeaderStatusMessage, message)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if len(params) > 0 {
		statusCode = params[0]
		w.WriteHeader(statusCode)
	}

	var err error
	if payload != nil {
		err = json.NewEncoder(w).Encode(payload)
	} else if len(message) > 0 {
		_, err = fmt.Fprintln(w, message)
	}

	LogError(err)
}

//parseUserInput tries to read the body and parse it into p. Returns true on success
func parseUserInput(config *models.ConfigStruct, w http.ResponseWriter, r *http.Request, p interface{}) bool {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, config.Webserver.MaxBodyLength))

	if LogError(err) || LogError(r.Body.Close()) {
		return false
	}

	return !handleAndSendError(json.Unmarshal(body, p), w, models.WrongInputFormatError, http.StatusUnprocessableEntity)
}

func handleAndSendError(err error, w http.ResponseWriter, message string, statusCode int) bool {
	if !LogError(err) {
		return false
	}
	sendError(err.Error(), w, message, statusCode)
	return true
}

func sendError(erre string, w http.ResponseWriter, message string, statusCode int) {
	sendResponse(w, models.ResponseError, message, nil, statusCode)
}

func sendServerError(w http.ResponseWriter) {
	sendError("internal server error", w, models.ServerError, http.StatusInternalServerError)
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

func isStructInvalid(x interface{}) bool {
	s := reflect.TypeOf(x)
	for i := s.NumField() - 1; i >= 0; i-- {
		e := reflect.ValueOf(x).Field(i)

		if hasEmptyValue(e) {
			return true
		}
	}
	return false
}

func hasEmptyValue(e reflect.Value) bool {
	switch e.Type().Kind() {
	case reflect.String:
		if e.String() == "" || strings.Trim(e.String(), " ") == "" {
			return true
		}
	case reflect.Array:
		for j := e.Len() - 1; j >= 0; j-- {
			isEmpty := hasEmptyValue(e.Index(j))
			if isEmpty {
				return true
			}
		}
	case reflect.Slice:
		return isStructInvalid(e)

	case
		reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer,
		reflect.Uint64, reflect.Uint, reflect.Uint8, reflect.Bool,
		reflect.Struct, reflect.Int64, reflect.Int:
		{
			return false
		}
	default:
		log.Error(e.Type().Kind(), e)
		return true
	}
	return false
}

//Returns true on error
func checkPayloadSizes(w http.ResponseWriter, maxPayloadSize uint, contents ...string) bool {
	for _, content := range contents {
		if uint(len(content)) > maxPayloadSize-1 {
			sendResponse(w, models.ResponseError, "Content too long!", nil, http.StatusRequestEntityTooLarge)
			return true
		}
	}
	return false
}

func checkInput(w http.ResponseWriter, request interface{}, contents ...string) bool {
	if isStructInvalid(request) {
		sendError("input missing", w, models.WrongInputFormatError, http.StatusUnprocessableEntity)
		return true
	}

	return checkPayloadSizes(w, constants.DefaultMaxPayloadSize, contents...)
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

func headerToString(headers http.Header) string {
	var sheaders string
	for k, v := range headers {
		sheaders += k + "=" + strings.Join(v, ";") + "\r\n"
	}
	return sheaders
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

func tokenFromBearerHeader(header string) string {
	return strings.TrimSpace(strings.ReplaceAll(header, "Bearer", ""))
}
