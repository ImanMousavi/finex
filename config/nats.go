package config

import (
	"os"

	"github.com/nats-io/nats.go"
)

var Nats *nats.Conn

func ConnectNats() error {
	nats, err := nats.Connect(os.Getenv("NATS_URL"))

	if err != nil {
		return err
	}

	Nats = nats
	return nil
}
