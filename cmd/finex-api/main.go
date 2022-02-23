package main

import (
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/routes"
)

func main() {
	if err := config.InitializeConfig(); err != nil {
		config.Logger.Error(err.Error())
		return
	}

	r := routes.SetupRouter()
	// running
	r.Listen(":3000")
}
