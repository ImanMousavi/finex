package config

import (
	"os"

	"github.com/nats-io/nats.go"
)

var Nats *nats.Conn

func ConnectNats() error {
	if len(os.Getenv("NATS_USER")) >= 0 && len(os.Getenv("NATS_PASS")) >= 0 {
		if n, err := nats.Connect(os.Getenv("NATS_URL"), nats.UserInfo(os.Getenv("NATS_USER"), os.Getenv("NATS_PASS"))); err != nil {
			return err
		} else {
			Nats = n
		}
	} else {
		if n, err := nats.Connect(os.Getenv("NATS_URL")); err != nil {
			return err
		} else {
			Nats = n
		}
	}

	return nil
}
