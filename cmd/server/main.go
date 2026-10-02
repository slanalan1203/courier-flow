package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	deliverypb "github.com/slanalan1203/courier-flow/api"
	postgresrepository "github.com/slanalan1203/courier-flow/internal/repository/postgres"
	deliveryservice "github.com/slanalan1203/courier-flow/internal/service"
	"google.golang.org/grpc"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	deliveryRepository :=
		postgresrepository.NewDeliveryRepository(pool)

	deliveryService :=
		deliveryservice.NewDeliveryService(deliveryRepository)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	deliverypb.RegisterDeliveryServiceServer(
		grpcServer,
		deliveryService,
	)

	log.Printf("server listening at %v", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
