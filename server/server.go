package server

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync/atomic"

	"gohw/shared/utils/dotenv"
	"golang.org/x/time/rate"
)

type SpamerRequest struct {
	Id int `json:"Id"`
}

type ClientStats struct {
	Positive int64 `json:"positive"`
	Negative int64 `json:"negative"`
}

type ServerStats struct {
	Total   ClientStats   `json:"total"`
	Clients []ClientStats `json:"clients"`
}

const (
	APP_SERVER_PORT = "APP_SERVER_PORT"
	APP_MAX_RPS     = 5
)

var (
	serverStats = ServerStats{
		Total:   ClientStats{},
		Clients: make([]ClientStats, 2),
	}
	limiter = rate.NewLimiter(rate.Limit(APP_MAX_RPS), APP_MAX_RPS)
)

func getHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serverStats)
}

func getRandomStatusCode() int {
	res := rand.Float64()
	if res < 0.7 {
		if res < 0.35 {
			return http.StatusOK
		}
		return http.StatusAccepted
	}
	if res < 0.85 {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func updateClientStats(clientStats *ClientStats, statusCode int) {
	if statusCode == http.StatusOK || statusCode == http.StatusAccepted {
		atomic.AddInt64(&clientStats.Positive, 1)
	} else {
		atomic.AddInt64(&clientStats.Negative, 1)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if !limiter.Allow() {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var body SpamerRequest
	if err := decoder.Decode(&body); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	statusCode := getRandomStatusCode()
	w.WriteHeader(statusCode)

	updateClientStats(&serverStats.Total, statusCode)
	updateClientStats(&serverStats.Clients[body.Id], statusCode)
}

func handler(w http.ResponseWriter, r *http.Request) {
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
	http.HandleFunc("/", handler)

	log.Printf("Starting server at port %s", port)
	http.ListenAndServe(":"+port, nil)
}