package main

import (
	"fmt"
	"os"
	"time"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/workers/engines"
)

func CreateWorker(id string) engines.Worker {
	switch id {
	case "matching":
		return engines.NewMatchingWorker()
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

		prefetch := mq_client.GetPrefetchCount(id)

		if prefetch > 0 {
			mq_client.GetChannel().Qos(prefetch, 0, false)
		}

		sub, _ := config.Nats.QueueSubscribeSync(id, id)

		for {
			m, err := sub.NextMsg(1 * time.Second)

			if err != nil {
				continue
			}

			config.Logger.Infof("Receive message: %s", string(m.Data))
			if err := worker.Process(m.Data); err == nil {
				m.Ack()
			} else {
				config.Logger.Errorf("Worker error: %v", err.Error())
			}
		}

	}
}
