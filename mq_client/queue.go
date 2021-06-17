package mq_client

import (
	"github.com/streadway/amqp"
)

var AMQPChannel *amqp.Channel
var Connection *amqp.Connection

func Connect() error {
	cn, err := CreateAMQP()
	if err != nil {
		return err
	}

	Connection = cn

	return nil
}

func GetChannel() *amqp.Channel {
	if AMQPChannel != nil {
		return AMQPChannel
	} else {
		channel, _ := Connection.Channel()

		AMQPChannel = channel

		return AMQPChannel
	}
}

func Publish(eid string, queue Queue, payload []byte, routing_key string) error {
	exchangeName, exchangeKind := GetExchange(eid)

	err := GetChannel().ExchangeDeclare(exchangeName, exchangeKind, queue.Durable, false, false, false, nil)

	if err != nil {
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

func EnqueueEvent(kind string, id string, event string, payload []byte) error {
	routing_key := kind + "." + id + "." + event

	GetChannel().ExchangeDeclare("peatio.events.ranger", "topic", false, false, false, false, nil)

	return GetChannel().Publish(
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

func ChanEnqueueEvent(channel *amqp.Channel, kind string, id string, event string, payload []byte) error {
	routing_key := kind + "." + id + "." + event

	channel.ExchangeDeclare("peatio.events.ranger", "topic", false, false, false, false, nil)

	return channel.Publish(
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
