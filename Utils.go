package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

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
	if dur > 1*time.Second {
		log.Printf("Header checking took %s\n", dur.String())
	}

	return false
}
