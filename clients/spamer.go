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
    N_REQUESTS = "N_REQUESTS"
    N_WORKERS = "N_WORKERS"
    N_WORKER_REQUESTS = "N_WORKER_REQUESTS"
    CLIENT_MAX_RPS = "CLIENT_MAX_RPS"
)

var (
    statsLock sync.Mutex
    stats = map[int]int{}

    nRequestsLeft int64

    limiter *rate.Limiter
)

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

func updateStats(statusCode int) {
    statsLock.Lock()
    _, ok := stats[statusCode]
    if ok {
        stats[statusCode] += 1
    } else {
        stats[statusCode] = 1
    }
    statsLock.Unlock()
}

func sendPostRequest(serverUrl string, data []byte) {
    reservation := limiter.Reserve()
    time.Sleep(reservation.Delay())

    resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(data))
    if err != nil {
        log.Fatalf("Error sending POST request: %v\n", err)
        return
    }
    defer resp.Body.Close()

    updateStats(resp.StatusCode)
    log.Printf("Response status: %s\n", resp.Status)
}

func runWorker(workersWg *sync.WaitGroup, nWorkerRequests int, serverUrl string, data []byte) {
    defer workersWg.Done()

    for {
        currentRequestsLeft := atomic.LoadInt64(&nRequestsLeft)
        if currentRequestsLeft <= 0 {
            return
        }

        newValue := int64(math.Max(0, float64(currentRequestsLeft - int64(nWorkerRequests))))
        swapped := atomic.CompareAndSwapInt64(&nRequestsLeft, currentRequestsLeft, newValue)

        if !swapped {
            continue
        }

        requestsTBD := currentRequestsLeft - newValue
        for i := int64(0); i < requestsTBD; i++ {
            sendPostRequest(serverUrl, data)
        }
    }
}

func getStatMessage() string {
    if len(stats) == 0 {
        return "Нет статистики"
    }

    var statuses []string
    statsLock.Lock()
    for k, v := range stats {
        statuses = append(statuses, fmt.Sprintf(" %d - %d", k, v))
    }
    statsLock.Unlock()

    return "Разбивка по статусам: " + strings.Join(statuses, ", ")
}

func RunSpamerClient(id int) {
    serverUrl, nRequests, nWorkerRequests, nWorkers, maxRps := loadEnvVars()

    nRequestsLeft = int64(nRequests)
    limiter = rate.NewLimiter(rate.Limit(maxRps), maxRps)

    data := []byte(fmt.Sprintf(`{"Id": %d}`, id))

    var workersWg sync.WaitGroup
    workersWg.Add(nWorkers)
    for i := 0; i < nWorkers; i++ {
        go runWorker(&workersWg, nWorkerRequests, serverUrl, data)
    }
    workersWg.Wait()

    log.Print(getStatMessage())
}
