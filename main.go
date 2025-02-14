package main

import (
	"gohw/clients"
	"gohw/server"
	"time"
)

func main() {
    go server.Run()
	go clients.RunCheckerClient()

	time.Sleep(time.Duration(time.Second))

	go clients.RunSpamerClient(0)
	go clients.RunSpamerClient(1)

	time.Sleep(10 * time.Duration(time.Second))
}
