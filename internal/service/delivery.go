package service

import (
	"cmp"
	"context"
	"io"
	"slices"
	"strings"
	"sync"

	deliverypb "github.com/slanalan1203/courier-flow/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeliveryService struct {
	deliverypb.UnimplementedDeliveryServiceServer

	mu         sync.RWMutex
	nextID     int64
	deliveries map[int64]*deliverypb.Delivery
}

func NewDeliveryService() *DeliveryService {
	return &DeliveryService{
		deliveries: make(map[int64]*deliverypb.Delivery),
	}
}

func (s *DeliveryService) CreateDelivery(
	ctx context.Context,
	req *deliverypb.CreateDeliveryRequest,
) (*deliverypb.Delivery, error) {
	address := strings.TrimSpace(req.GetAddress())
	if address == "" {
		return nil, status.Error(codes.InvalidArgument, "empty address")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++

	delivery := &deliverypb.Delivery{
		Id:      s.nextID,
		Address: address,
		Status:  deliverypb.DeliveryStatus_DELIVERY_STATUS_CREATED,
	}

	s.deliveries[s.nextID] = delivery

	return delivery, nil
}

func (s *DeliveryService) GetDelivery(
	ctx context.Context,
	req *deliverypb.GetDeliveryRequest,
) (*deliverypb.Delivery, error) {
	id := req.GetId()

	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	delivery, ok := s.deliveries[id]
	if !ok {
		return nil, status.Error(codes.NotFound, "delivery not found")
	}
	return &deliverypb.Delivery{
		Id:      delivery.Id,
		Address: delivery.Address,
		Status:  delivery.Status,
	}, nil
}

func (s *DeliveryService) UpdateDeliveryStatus(
	ctx context.Context,
	req *deliverypb.UpdateDeliveryStatusRequest) (*deliverypb.Delivery, error) {
	id := req.GetId()
	newStatus := req.GetStatus()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	if newStatus == deliverypb.DeliveryStatus_DELIVERY_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "status must be specified")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delivery, ok := s.deliveries[id]
	if !ok {
		return nil, status.Error(codes.NotFound, "delivery not found")
	}
	if !canTransition(delivery.GetStatus(), newStatus) {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"cannot change status from %s to %s",
			delivery.GetStatus(),
			newStatus,
		)
	}
	delivery.Status = newStatus
	return &deliverypb.Delivery{
		Id:      delivery.GetId(),
		Address: delivery.GetAddress(),
		Status:  delivery.GetStatus(),
	}, nil
}

func (s *DeliveryService) ListDelivery(
	req *deliverypb.ListDeliveryRequest,
	stream deliverypb.DeliveryService_ListDeliveryServer) error {
	s.mu.RLock()
	deliveries := make([]*deliverypb.Delivery, 0, len(s.deliveries))
	for _, delivery := range s.deliveries {
		deliveries = append(deliveries, &deliverypb.Delivery{
			Id:      delivery.GetId(),
			Address: delivery.GetAddress(),
			Status:  delivery.GetStatus(),
		})
	}
	s.mu.RUnlock()
	slices.SortFunc(deliveries, func(i, j *deliverypb.Delivery) int {
		return cmp.Compare(i.GetId(), j.GetId())
	})
	for _, delivery := range deliveries {
		if err := stream.Send(delivery); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeliveryService) ReportLocation(
	stream deliverypb.DeliveryService_ReportLocationServer) error {
	var deliveryID int64
	var pointReceived int32

	for {
		point, err := stream.Recv()
		if err == io.EOF {
			if pointReceived == 0 {
				return status.Error(codes.InvalidArgument, "no point received")
			}
			return stream.SendAndClose(&deliverypb.LocationReport{
				DeliveryId:     deliveryID,
				PointsReceived: pointReceived,
			})
		}
		if err != nil {
			return err
		}

		id := point.GetDeliveryId()
		if id <= 0 {
			return status.Error(codes.InvalidArgument, "invalid id")
		}
		if pointReceived == 0 {
			s.mu.RLock()
			_, exists := s.deliveries[id]
			s.mu.RUnlock()
			if !exists {
				return status.Error(codes.NotFound, "delivery not found")
			}
			deliveryID = id
		} else if deliveryID != id {
			return status.Error(codes.InvalidArgument, "delivery id mismatch")
		}
		pointReceived++
	}
}

func (s *DeliveryService) TrackDelivery(
	stream deliverypb.DeliveryService_TrackDeliveryServer) error {
	var deliveryID int64
	var acceptedPoints int32

	for {
		point, err := stream.Recv()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}
		id := point.GetDeliveryId()
		if id <= 0 {
			return status.Error(codes.InvalidArgument, "invalid id")
		}
		if acceptedPoints == 0 {
			s.mu.RLock()
			_, exists := s.deliveries[id]
			s.mu.RUnlock()
			if !exists {
				return status.Error(codes.NotFound, "delivery not found")
			}
			deliveryID = id
		} else if deliveryID != id {
			return status.Error(codes.InvalidArgument, "delivery id mismatch")
		}
		acceptedPoints++

		err = stream.Send(&deliverypb.LocationAck{
			DeliveryId: deliveryID,
			AcceptedPoints: acceptedPoints,
		})
		if err != nil {
			return err
		}
	}
}

func canTransition(
	from deliverypb.DeliveryStatus,
	to deliverypb.DeliveryStatus) bool {
	switch from {
	case deliverypb.DeliveryStatus_DELIVERY_STATUS_CREATED:
		return to == deliverypb.DeliveryStatus_DELIVERY_STATUS_PICKED_UP
	case deliverypb.DeliveryStatus_DELIVERY_STATUS_PICKED_UP:
		return to == deliverypb.DeliveryStatus_DELIVERY_STATUS_IN_TRANSIT
	case deliverypb.DeliveryStatus_DELIVERY_STATUS_IN_TRANSIT:
		return to == deliverypb.DeliveryStatus_DELIVERY_STATUS_DELIVERED
	default:
		return false
	}
}
