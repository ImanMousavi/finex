package main

import (
	"fmt"
	"os"
	"time"

	"github.com/streadway/amqp"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/workers/engines"
)

var Queue = &[]amqp.Queue{}
var Connection *amqp.Connection

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

	Connection = mq_client.Connection
	Channel := mq_client.GetChannel()

	ARVG := os.Args[1:]

	for _, id := range ARVG {
		fmt.Println("Start finex-engine: " + id)
		worker := CreateWorker(id)

		prefetch := mq_client.GetPrefetchCount(id)

		if prefetch > 0 {
			mq_client.GetChannel().Qos(prefetch, 0, false)
		}

		binding_queue := mq_client.GetBindingQueue(id)
		binding_queue_id := mq_client.GetBindingExchangeId(id)
		exchange_name, exchange_kind := mq_client.GetExchange(binding_queue_id)
		routing_key := mq_client.GetRoutingKey(id)

		if err := Channel.ExchangeDeclare(exchange_name, exchange_kind, binding_queue.Durable, false, false, false, nil); err != nil {
			config.Logger.Errorf("Exchange Declare: %v\n", err)
			return
		}
		if _, err := Channel.QueueDeclare(binding_queue.Name, binding_queue.Durable, false, false, false, nil); err != nil {
			config.Logger.Errorf("Queue Declare: %v\n", err)
			return
		}
		Channel.QueueBind(binding_queue.Name, routing_key, exchange_name, false, nil)

		sub, _ := config.Nats.QueueSubscribeSync(id, binding_queue.Name)

		for {
			m, err := sub.NextMsg(1 * time.Second)

			if err != nil {
				continue
			}

			// config.Logger.Infof("Receive message: %s", string(m.Data))
			if err := worker.Process(m.Data); err == nil {
				m.Ack()
			} else {
				config.Logger.Errorf("Worker error: %v", err.Error())
			}
		}

	}
}
