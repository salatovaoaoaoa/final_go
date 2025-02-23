package clients

import (
	"bytes"
	"fmt"
	"gohw/clients/utils/url"
	"gohw/shared/utils/dotenv"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const (
	N_REQUESTS        = "N_REQUESTS"
	N_WORKERS         = "N_WORKERS"
	N_WORKER_REQUESTS = "N_WORKER_REQUESTS"
	CLIENT_MAX_RPS    = "CLIENT_MAX_RPS"
)

type Spamer struct {
	StatsLock *sync.Mutex
	Stats     map[int]int

	NRequestsLeft int64

	Limiter *rate.Limiter
}

func loadEnvVars() (
	serverUrl string,
	nRequests int,
	nWorkerRequests int,
	nWorkers int,
	maxRps int,
) {
	err := dotenv.LoadEnv()
	if err != nil {
		log.Fatal("Error loading env file")
	}

	serverUrl = url.GetServerUrl()
	nRequests, err = strconv.Atoi(dotenv.GetEnvVar(N_REQUESTS))

	if err != nil {
		log.Printf("Invalid %s env parameter\n", N_REQUESTS)
		return
	}

	nWorkers, err = strconv.Atoi(dotenv.GetEnvVar(N_WORKERS))

	if err != nil {
		log.Printf("Invalid %s env parameter\n", N_WORKERS)
		return
	}

	nWorkerRequests, err = strconv.Atoi(dotenv.GetEnvVar(N_WORKER_REQUESTS))

	if err != nil {
		log.Printf("Invalid %s env parameter\n", N_WORKER_REQUESTS)
		return
	}

	maxRps, err = strconv.Atoi(dotenv.GetEnvVar(CLIENT_MAX_RPS))
	if err != nil {
		log.Printf("Invalid %s env parameter\n", CLIENT_MAX_RPS)
		return
	}

	return
}

func (s Spamer) updateStats(statusCode int) {
	s.StatsLock.Lock()
	_, ok := s.Stats[statusCode]
	if ok {
		s.Stats[statusCode] += 1
	} else {
		s.Stats[statusCode] = 1
	}
	s.StatsLock.Unlock()
}

func (s Spamer) sendPostRequest(serverUrl string, data []byte) {
	reservation := s.Limiter.Reserve()
	time.Sleep(reservation.Delay())

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Error sending POST request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	s.updateStats(resp.StatusCode)
	log.Printf("Response status: %s\n", resp.Status)
}

func (s Spamer) runWorker(workersWg *sync.WaitGroup, nWorkerRequests int, serverUrl string, data []byte) {
	defer workersWg.Done()

	for {
		currentRequestsLeft := atomic.LoadInt64(&s.NRequestsLeft)
		if currentRequestsLeft <= 0 {
			return
		}

		newValue := int64(math.Max(0, float64(currentRequestsLeft-int64(nWorkerRequests))))
		swapped := atomic.CompareAndSwapInt64(&s.NRequestsLeft, currentRequestsLeft, newValue)

		if !swapped {
			continue
		}

		requestsTBD := currentRequestsLeft - newValue
		for i := int64(0); i < requestsTBD; i++ {
			s.sendPostRequest(serverUrl, data)
		}
	}
}

func (s Spamer) getStatMessage() string {
	if len(s.Stats) == 0 {
		return "Нет статистики"
	}

	var statuses []string
	s.StatsLock.Lock()
	for k, v := range s.Stats {
		statuses = append(statuses, fmt.Sprintf(" %d - %d", k, v))
	}
	s.StatsLock.Unlock()

	return "Разбивка по статусам: " + strings.Join(statuses, ", ")
}

func (s Spamer) run(id int, serverUrl string, nWorkers int, nWorkerRequests int) {
	data := []byte(fmt.Sprintf(`{"Id": %d}`, id))

	var workersWg sync.WaitGroup
	workersWg.Add(nWorkers)
	for i := 0; i < nWorkers; i++ {
		go s.runWorker(&workersWg, nWorkerRequests, serverUrl, data)
	}
	workersWg.Wait()

	log.Print(s.getStatMessage())
}

func RunSpamerClient(id int) {
	serverUrl, nRequests, nWorkerRequests, nWorkers, maxRps := loadEnvVars()

	spamer := Spamer{
		StatsLock:     &sync.Mutex{},
		Stats:         map[int]int{},
		NRequestsLeft: int64(nRequests),
		Limiter:       rate.NewLimiter(rate.Limit(maxRps), maxRps),
	}

	spamer.run(id, serverUrl, nWorkers, nWorkerRequests)
}