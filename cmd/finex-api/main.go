package main

import (
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/routes"
)

func main() {
	if err := config.InitializeConfig(); err != nil {
		config.Logger.Error(err.Error())
		return
	}
	if err := mq_client.Connect(); err != nil {
		config.Logger.Error(err.Error())
		return
	}

	r := routes.SetupRouter()
	// running
	r.Listen(":3000")
}
