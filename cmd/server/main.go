package main

import (
	"log"
	"net"

	deliverypb "github.com/slanalan1203/courier-flow/api"
	deliveryservice "github.com/slanalan1203/courier-flow/internal/service"
	"google.golang.org/grpc"
)

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	deliverypb.RegisterDeliveryServiceServer(grpcServer, deliveryservice.NewDeliveryService())

	log.Printf("server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
