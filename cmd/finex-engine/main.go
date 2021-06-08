package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/mq_client"
	"github.com/zsmartex/go-finex/workers/engines"
)

var Queue = &[]amqp.Queue{}
var Connection *amqp.Connection

func randomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}

func CreateWorker(id string) engines.Worker {
	switch id {
	case "matching":
		return engines.NewMatchingWorker()
	case "order_processor":
		return engines.NewOrderProcessorWorker()
	case "trade_executor":
		return engines.NewTradeExecutorWorker()
	case "depth_cache":
		return engines.NewDeptCachehWorker()
	default:
		return nil
	}
}

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := mq_client.Connect(); err != nil {
		fmt.Println(err.Error())
		return
	}

	// done := make(chan error)

	Connection = mq_client.Connection
	Channel := mq_client.GetChannel()

	ARVG := os.Args[1:]

	for _, id := range ARVG {
		fmt.Println("Start finex-engine: " + id)
		worker := CreateWorker(id)

		consumer_tag := randomString(16)

		prefetch := mq_client.GetPrefetchCount(id)

		if prefetch > 0 {
			mq_client.GetChannel().Qos(prefetch, 0, false)
		}

		binding_queue := mq_client.GetBindingQueue(id)
		binding_queue_id := mq_client.GetBindingExchangeId(id)
		exchange_name, exchange_kind := mq_client.GetExchange(binding_queue_id)
		routing_key := mq_client.GetRoutingKey(id)

		if err := Channel.ExchangeDeclare(exchange_name, exchange_kind, binding_queue.Durable, false, false, false, nil); err != nil {
			log.Fatalf("Exchange Declare: %v\n", err)
			return
		}
		// Channel.ExchangeBind(destination string, key string, source string, noWait bool, args amqp.Table)
		if _, err := Channel.QueueDeclare(binding_queue.Name, binding_queue.Durable, false, false, false, nil); err != nil {
			log.Fatalf("Queue Declare: %v\n", err)
			return
		}
		Channel.QueueBind(binding_queue.Name, routing_key, exchange_name, false, nil)

		deliveries, err := Channel.Consume(
			binding_queue.Name,
			consumer_tag,
			false,
			false,
			false,
			false,
			nil,
		)

		if err != nil {
			log.Printf("Queue Consume: %v", err)
			continue
		}

		for d := range deliveries {
			log.Printf("Receive message: %s\n", string(d.Body))
			worker.Process(d.Body)
			d.Ack(false)
		}

		// go HandleMessage(worker, deliveries, done)
	}
}

// func HandleMessage(worker workers.Worker, deliveries <-chan amqp.Delivery, done chan error) {
// 	for msg := range deliveries {
// 		worker.Process(msg.Body)
// 	}

// 	log.Printf("handle: deliveries channel closed")
// 	done <- nil
// }
