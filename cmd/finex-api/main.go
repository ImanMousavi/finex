package main

import (
	"fmt"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/mq_client"
	"github.com/zsmartex/go-finex/routes"
)

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := mq_client.Connect(); err != nil {
		fmt.Println(err.Error())
		return
	}

	r := routes.SetupRouter()
	// running
	r.Listen(":3000")
}
