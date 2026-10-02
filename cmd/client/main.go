package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	deliverypb "github.com/slanalan1203/courier-flow/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := deliverypb.NewDeliveryServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	delivery, err := client.CreateDelivery(
		ctx,
		&deliverypb.CreateDeliveryRequest{
			Address: "Москва, Тверская 1",
		})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(
		"Create delivery successfully",
		delivery.GetId(),
		delivery.GetAddress(),
		delivery.GetStatus())

	found, err := client.GetDelivery(ctx, &deliverypb.GetDeliveryRequest{
		Id: delivery.GetId(),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(
		"Get delivery successfully",
		found.GetId(),
		found.GetAddress(),
		found.GetStatus())
	updated, err := client.UpdateDeliveryStatus(
		ctx, &deliverypb.UpdateDeliveryStatusRequest{
			Id:     delivery.GetId(),
			Status: deliverypb.DeliveryStatus_DELIVERY_STATUS_PICKED_UP,
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(
		"Update delivery successfully",
		updated.GetId(),
		updated.GetAddress(),
		updated.GetStatus())

	isTransit, err := client.UpdateDeliveryStatus(
		ctx, &deliverypb.UpdateDeliveryStatusRequest{
			Id:     delivery.GetId(),
			Status: deliverypb.DeliveryStatus_DELIVERY_STATUS_IN_TRANSIT,
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(
		"delivery in transit",
		isTransit.GetId(),
		isTransit.GetAddress(),
		isTransit.GetStatus())
	deivered, err := client.UpdateDeliveryStatus(
		ctx, &deliverypb.UpdateDeliveryStatusRequest{
			Id:     delivery.GetId(),
			Status: deliverypb.DeliveryStatus_DELIVERY_STATUS_DELIVERED,
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(
		"delivery in complit",
		deivered.GetId(),
		deivered.GetAddress(),
		deivered.GetStatus())

	deliveryStream, err := client.ListDelivery(ctx, &deliverypb.ListDeliveryRequest{})
	if err != nil {
		log.Fatal(err)
	}
	for {
		item, err := deliveryStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(
			"delivery in stream",
			item.GetId(),
			item.GetAddress(),
			item.GetStatus())
	}

	locationStream, err := client.ReportLocation(ctx)
	if err != nil {
		log.Fatal(err)
	}
	points := []*deliverypb.LocationPoint{
		{
			DeliveryId: delivery.GetId(),
			Latitude:   55.751244,
			Longitude:  37.618423,
		},
		{
			DeliveryId: delivery.GetId(),
			Latitude:   55.752500,
			Longitude:  37.620000,
		},
		{
			DeliveryId: delivery.GetId(),
			Latitude:   55.754000,
			Longitude:  37.623000,
		},
	}
	for _, point := range points {
		if err := locationStream.Send(point); err != nil {
			log.Fatal(err)
		}
	}
	report, err := locationStream.CloseAndRecv()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(
		"location report",
		report.GetDeliveryId(),
		report.GetPointsReceived())

	trackingStream, err := client.TrackDelivery(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, point := range points {
		if err := trackingStream.Send(point); err != nil {
			log.Fatal(err)
		}
		ack, err := trackingStream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(
			"tracking ack",
			ack.GetDeliveryId(),
			ack.GetAcceptedPoints())
	}
	if err := trackingStream.CloseSend(); err != nil {
		log.Fatal(err)
	}
	_, err = trackingStream.Recv()
	if err != io.EOF {
		log.Fatal("tracking stream closed unexpectedly")
	}
}
