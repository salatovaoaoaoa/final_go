package server

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	"gohw/shared/utils/dotenv"

	"golang.org/x/time/rate"
)

type SpamerRequest struct {
    Id int
}

type ClientStats struct {
    Positive int64
    Negative int64
}

type ServerStats struct {
    Total ClientStats
    Clients map[int]ClientStats
}

const (
    APP_SERVER_PORT = "APP_SERVER_PORT"
    APP_MAX_RPS = "APP_MAX_RPS"
)

var (
    statsLock sync.Mutex
    serverStats = ServerStats{
        Total: ClientStats{
            Positive: 0,
            Negative: 0,
        },
        Clients: map[int]ClientStats{},
    }

    limiter *rate.Limiter

    positiveStatuses = []int{http.StatusOK, http.StatusAccepted}
    negativeStatuses = []int{http.StatusBadRequest, http.StatusInternalServerError}
)

func getHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(serverStats)
}

func getRandomStatusCode() int {
    res := rand.Float64()
    if (res < 0.7) {
        if (res < 0.35) {
            return http.StatusOK
        } else {
            return http.StatusAccepted
        }
    } else {
        if (res < 0.85) {
            return http.StatusBadRequest
        } else {
            return http.StatusInternalServerError
        }
    }
}

func updateClientStats(clientStats ClientStats, statusCode int) {
    if slices.Contains(positiveStatuses, statusCode) {
        atomic.AddInt64(&clientStats.Positive, 1)
    }

    if slices.Contains(negativeStatuses, statusCode) {
        atomic.AddInt64(&clientStats.Negative, 1)
    }
}

func updateServerStats(id int, statusCode int) {
    updateClientStats(serverStats.Total, statusCode)

    for {
        _, ok := serverStats.Clients[id]
        if ok {
            break
        }
        statsLock.Lock()

        _, ok = serverStats.Clients[id]

        if !ok {
            serverStats.Clients[id] = ClientStats{
                Positive: 0,
                Negative: 0,
            }
        }
        
        statsLock.Unlock()
    }

    updateClientStats(serverStats.Clients[id], statusCode)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    var body SpamerRequest
    err := decoder.Decode(&body)
    if err != nil {
        log.Fatal(err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    statusCode := getRandomStatusCode()
    w.WriteHeader(statusCode)
    updateServerStats(body.Id, statusCode)
}

func handler(w http.ResponseWriter, r *http.Request) {
    if !limiter.Allow() {
        http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
        return
    }

    switch r.Method {
    case http.MethodGet:
        getHandler(w, r)
    case http.MethodPost:
        postHandler(w, r)
    }
}

func Run() {
    err := dotenv.LoadEnv()
	if err != nil {
		log.Fatal("Error loading env file")
	}

	port := dotenv.GetEnvVar(APP_SERVER_PORT)

    maxRps, err := strconv.Atoi(dotenv.GetEnvVar(APP_MAX_RPS))
    if err != nil {
        log.Fatal("Invalid APP_MAX_RPS env parameter")
        return
    }
    limiter = rate.NewLimiter(rate.Limit(maxRps), maxRps)

	http.HandleFunc("/", handler)

	log.Printf("Starting server at port %s\n", port)
    http.ListenAndServe(":" + port, nil)
}
