package main

import (
	"fmt"

	"gitlab.com/zsmartex/finex/config"
	"gitlab.com/zsmartex/finex/routes"
)

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	r := routes.SetupRouter()
	// running
	r.Listen(":3000")
}
