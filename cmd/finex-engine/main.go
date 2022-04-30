package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/workers/engines"
	"github.com/zsmartex/pkg/services"
)

func CreateWorker(id string) engines.Worker {
	switch id {
	case "order_processor":
		return engines.NewOrderProcessorWorker()
	case "trade_executor":
		return engines.NewTradeExecutorWorker()
	case "ieo_order_processor":
		return engines.NewIEOOrderProcessorWorker()
	case "ieo_order_executor":
		return engines.NewIEOOrderExecutorWorker()
	default:
		return nil
	}
}

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	ARVG := os.Args[1:]
	id := ARVG[0]
	consumer, err := services.NewKafkaConsumer(strings.Split(os.Getenv("KAFKA_URL"), ""), "zsmartex", []string{id})
	if err != nil {
		panic(err)
	}

	fmt.Println("Start finex-engine: " + id)
	worker := CreateWorker(id)

	defer consumer.Close()

	for {
		records, err := consumer.Poll()
		if err != nil {
			config.Logger.Fatalf("Failed to poll consumer %v", err)
		}

		for _, record := range records {
			err := worker.Process(record.Value)

			if err != nil {
				config.Logger.Errorf("Worker error: %v", err.Error())
			}

			consumer.CommitRecords(*record)
		}
	}
}
