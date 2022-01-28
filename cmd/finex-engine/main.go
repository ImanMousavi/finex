package main

import (
	"fmt"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/workers/engines"
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
	mq_client.Connect()

	ARVG := os.Args[1:]

	for _, id := range ARVG {
		fmt.Println("Start finex-engine: " + id)
		worker := CreateWorker(id)

		config.Kafka.Subscribe(id, func(c *kafka.Consumer, e kafka.Event) error {
			config.Logger.Infof("Receive message: %s", e.String())

			err := worker.Process([]byte(e.String()))

			if err != nil {
				config.Logger.Errorf("Worker error: %v", err.Error())
			}

			return err
		})
	}
}
