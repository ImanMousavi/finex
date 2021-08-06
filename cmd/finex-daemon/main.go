package main

import (
	"fmt"
	"os"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/workers/daemons"
)

func CreateWorker(id string) daemons.Worker {
	switch id {
	case "cron_job":
		return daemons.NewCronJob()
	default:
		return nil
	}
}

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}
	mq_client.Connect()

	ARVG := os.Args[1:]

	for _, id := range ARVG {
		fmt.Println("Start finex-daemon: " + id)
		worker := CreateWorker(id)

		worker.Start()
	}
}
