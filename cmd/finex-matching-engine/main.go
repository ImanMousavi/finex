package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	GrpcEngine "github.com/zsmartex/pkg/Grpc/engine"
	"github.com/zsmartex/pkg/services"
	"google.golang.org/grpc"

	"github.com/zsmartex/finex/config"
	engine "github.com/zsmartex/finex/server"
)

func main() {
	if err := config.InitializeConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	server := engine.NewEngineServer()
	grpcServer := grpc.NewServer()

	consumer, err := services.NewKafkaConsumer(strings.Split(os.Getenv("KAFKA_URL"), ","), "zsmartex", []string{"matching"})
	if err != nil {
		panic(err)
	}

	defer consumer.Close()

	go func() {
		for {

			records, err := consumer.Poll()
			if err != nil {
				config.Logger.Fatalf("Failed to poll consumer %v", err)
			}

			for _, record := range records {
				if record.Topic != "matching" {
					continue
				}

				config.Logger.Debugf("Recevie message from topic: %s payload: %s", record.Topic, string(record.Value))
				err := server.Process(record.Value)

				if err != nil {
					config.Logger.Fatalf("Worker error: %v", err.Error())
				}

				consumer.CommitRecords(*record)
			}
		}
	}()

	config.Logger.Info("Starting Finex G-RPC")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("ENGINE_PORT")))
	if err != nil {
		config.Logger.Fatalf("failed to listen: %v", err)
	}

	GrpcEngine.RegisterMatchingEngineServiceServer(grpcServer, server)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
