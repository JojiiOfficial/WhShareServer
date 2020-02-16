package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strings"

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
