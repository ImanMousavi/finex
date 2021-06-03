package mq_client

import "github.com/streadway/amqp"

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
		AMQPChannel, _ := Connection.Channel()

		return AMQPChannel
	}
}

func Publish(eid string, payload []byte, routing_key string) error {
	_n, _t := GetExchange(eid)

	err := GetChannel().ExchangeDeclare(_n, _t, false, false, false, false, nil)

	if err != nil {
		return err
	}

	GetChannel().Publish(
		eid,
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

	return nil
}

func Enqueue(id string, payload []byte) {
	eid := GetBindingExchangeId(id)
	routing_key := GetRoutingKey(id)

	Publish(eid, payload, routing_key)
}

func EnqueueEvent(kind string, id string, event string, payload []byte) {
	routing_key := kind + "." + "id" + "." + "event"

	GetChannel().ExchangeDeclare("peatio.events.ranger", "topic", false, false, false, false, nil)
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
