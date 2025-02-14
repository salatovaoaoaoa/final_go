package clients

import (
	"encoding/json"
	"fmt"
	"gohw/clients/utils/url"
	"gohw/shared/utils/dotenv"
	"log"
	"net/http"
	"strings"
	"time"
)

type ClientStats struct {
    Positive int64
    Negative int64
}

type ServerStats struct {
    Total ClientStats
    Clients map[int]ClientStats
}

func getStatsMessage(stats ServerStats) string {
    var parts = []string{fmt.Sprintf(
        "Total success %d, failed %d", stats.Total.Positive, stats.Total.Negative,
        )}

    for k, v := range stats.Clients {
        parts = append(parts, fmt.Sprintf(
            "Client %d success %d, failed %d", k, v.Positive, v.Negative,
        ))
    }

    return strings.Join(parts, "; ")
}

func RunCheckerClient() {
    err := dotenv.LoadEnv()
	if err != nil {
		log.Fatal("Error loading env file")
	}

    url := url.GetServerUrl()

    for {
        r, err := http.Get(url)
        if err != nil {
            log.Printf("Error sending GET request: %v", err)
        } else {
            decoder := json.NewDecoder(r.Body)
            var body ServerStats
            err := decoder.Decode(&body)
            if err != nil {
                log.Fatal(err)
            } else {
                log.Print(getStatsMessage(body))
            }
        }

        time.Sleep(5 * time.Second)
    }
}
