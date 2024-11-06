package internal

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func GetPage(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	body, _ = io.ReadAll(resp.Body)

	pause, _ := strconv.Atoi(os.Getenv("GOSPORTS_REQ_PAUSE"))
	if pause > 0 {
		log.Printf("sleeping %d seconds", pause)
		time.Sleep(time.Duration(pause) * time.Second)
	}

	return
}
