package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/zsmartex/finex/config"
	engine "github.com/zsmartex/finex/server"
	GrpcEngine "github.com/zsmartex/pkg/Grpc/engine"

	"google.golang.org/grpc"
)

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	server := engine.NewEngineServer()
	grpcServer := grpc.NewServer()

	config.Kafka.Subscribe("matching", func(c *kafka.Consumer, e kafka.Event) error {
		config.Logger.Infof("Receive message: %s", e.String())

		err := server.Process([]byte(e.String()))

		if err != nil {
			config.Logger.Errorf("Worker error: %v", err.Error())
		}

		return err
	})

	config.Logger.Println("Starting Finex G-RPC")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("ENGINE_PORT")))
	if err != nil {
		config.Logger.Fatalf("failed to listen: %v", err)
	}

	GrpcEngine.RegisterMatchingEngineServiceServer(grpcServer, server)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
