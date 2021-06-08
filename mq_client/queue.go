package mq_client

import (
	"log"

	"github.com/streadway/amqp"
	"github.com/zsmartex/go-finex/pkg/rabbitmq"
)

var AMQPChannel *rabbitmq.Channel
var Connection *rabbitmq.Connection

func Connect() error {
	cn, err := CreateAMQP()
	if err != nil {
		return err
	}

	Connection = cn

	return nil
}

func GetChannel() *rabbitmq.Channel {
	if AMQPChannel != nil {
		return AMQPChannel
	} else {
		AMQPChannel, err := Connection.Channel()

		if err != nil {
			log.Println("AMQP: Failed to get channel")
			log.Panic(err)
		}

		return AMQPChannel
	}
}

func Publish(eid string, queue Queue, payload []byte, routing_key string) error {
	exchangeName, exchangeKind := GetExchange(eid)

	err := GetChannel().ExchangeDeclare(exchangeName, exchangeKind, queue.Durable, false, false, false, nil)

	if err != nil {
		log.Println(err)
		return err
	}

	GetChannel().Publish(
		exchangeName,
		routing_key,
		false,
		false,
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "application/json",
			ContentEncoding: "",
			Body:            payload,
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	)

	return nil
}

func Enqueue(id string, payload []byte) {
	eid := GetBindingExchangeId(id)
	routing_key := GetRoutingKey(id)
	queue := GetBindingQueue(id)

	Publish(eid, queue, payload, routing_key)
}

func EnqueueEvent(kind string, id string, event string, payload []byte) {
	routing_key := kind + "." + id + "." + event

	GetChannel().ExchangeDeclare("peatio.events.ranger", "topic", false, false, false, false, nil)

	log.Printf("Publishing message to rango routing_key: %s\n", routing_key)

	GetChannel().Publish(
		"peatio.events.ranger",
		routing_key,
		false,
		false,
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "application/json",
			ContentEncoding: "",
			Body:            payload,
			DeliveryMode:    amqp.Persistent, // 1=non-persistent, 2=persistent
			Priority:        0,               // 0-9
			// a bunch of application/implementation-specific fields
		},
	)
}
