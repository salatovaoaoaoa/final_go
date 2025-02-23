package clients

import (
	"gohw/clients/utils/url"
	"gohw/shared/utils/dotenv"
	"log"
	"net/http"
	"time"
)

func RunCheckerClient() {
	err := dotenv.LoadEnv()
	if err != nil {
		log.Fatal("Error loading env file")
	}

	serverUrl := url.GetServerUrl()

	for {
		r, err := http.Get(serverUrl)
		if err != nil {
			log.Printf("Error sending GET request: %v", err)
		} else {
			log.Printf("Server status: %s", r.Status)
		}

		time.Sleep(5 * time.Second)
	}
}