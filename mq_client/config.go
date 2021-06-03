package mq_client

import (
	"io/ioutil"
	"os"
	"reflect"

	"github.com/streadway/amqp"
	"gopkg.in/yaml.v2"
)

var AMQPCfg *MQClientConfig

func CreateAMQP() (*amqp.Connection, error) {
	if err := LoadConfig(); err != nil {
		return nil, err
	}

	rabbitmq_username := os.Getenv("RABBITMQ_USERNAME")
	rabbitmq_password := os.Getenv("RABBITMQ_PASSWORD")
	rabbitmq_host := os.Getenv("RABBITMQ_HOST")
	rabbitmq_port := os.Getenv("RABBITMQ_PORT")

	connection, err := amqp.Dial("amqp://" + rabbitmq_username + ":" + rabbitmq_password + "@" + rabbitmq_host + ":" + rabbitmq_port)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func LoadConfig() error {
	buf, err := ioutil.ReadFile("config/amqp.yml")

	if err != nil {
		return err
	}

	c := &MQClientConfig{}

	err = yaml.Unmarshal(buf, c)

	if err != nil {
		return err
	}

	AMQPCfg = c

	return nil
}

func GetPrefetchCount(channel string) int {
	_channel := FindElementStruct(&AMQPCfg.Channel, "yaml", channel)

	if _channel != nil {
		return _channel.(Channel).Prefetch
	}

	return 0
}

func GetBindingExchangeId(id string) string {
	var exchange string
	binding := FindElementStruct(&AMQPCfg.Binding, "yaml", id)

	if binding != nil {
		return binding.(Binding).Exchange
	}

	return exchange
}

func GetBindingQueue(id string) Queue {
	queue_id := FindElementStruct(&AMQPCfg.Binding, "yaml", id).(Binding).Queue
	queue := FindElementStruct(&AMQPCfg.Queue, "yaml", queue_id).(Queue)
	return queue
}

func GetRoutingKey(id string) string {
	return GetBindingQueue(id).Name
}

func GetExchange(id string) (string, string) {
	exchange := FindElementStruct(&AMQPCfg.Exchange, "yaml", id).(Exchange)

	return exchange.Name, exchange.Type
}

func FindElementStruct(i interface{}, tag_name string, tag_value string) interface{} {
	e := reflect.ValueOf(i).Elem()

	for i := 0; i < e.NumField(); i++ {
		valueField := e.Field(i)
		typeField := e.Type().Field(i)
		Tag := typeField.Tag

		if tag_value == Tag.Get(tag_name) {
			return valueField.Interface()
		}
	}

	return nil
}
