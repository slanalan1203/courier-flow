package service

import (
	"context"
	"errors"
	"io"
	"strings"

	deliverypb "github.com/slanalan1203/courier-flow/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeliveryService struct {
	deliverypb.UnimplementedDeliveryServiceServer

	repository DeliveryRepository
}

func NewDeliveryService(repository DeliveryRepository) *DeliveryService {
	return &DeliveryService{
		repository: repository,
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

	delivery, err := s.repository.Create(
		ctx, address, deliverypb.DeliveryStatus_DELIVERY_STATUS_CREATED)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

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

	delivery, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, repositoryError(err)
	}
	return delivery, nil
}

func (s *DeliveryService) UpdateDeliveryStatus(
	ctx context.Context,
	req *deliverypb.UpdateDeliveryStatusRequest,
) (*deliverypb.Delivery, error) {
	id := req.GetId()
	newStatus := req.GetStatus()

	if id <= 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"invalid id",
		)
	}

	if newStatus == deliverypb.DeliveryStatus_DELIVERY_STATUS_UNSPECIFIED {
		return nil, status.Error(
			codes.InvalidArgument,
			"status must be specified",
		)
	}

	delivery, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, repositoryError(err)
	}

	if !canTransition(delivery.GetStatus(), newStatus) {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"cannot change status from %s to %s",
			delivery.GetStatus(),
			newStatus,
		)
	}

	updated, err := s.repository.UpdateStatus(
		ctx,
		id,
		newStatus,
	)
	if err != nil {
		return nil, repositoryError(err)
	}

	return updated, nil
}

func (s *DeliveryService) ListDelivery(
	req *deliverypb.ListDeliveryRequest,
	stream deliverypb.DeliveryService_ListDeliveryServer,
) error {
	deliveries, err := s.repository.List(stream.Context())
	if err != nil {
		return repositoryError(err)
	}

	for _, delivery := range deliveries {
		if err := stream.Send(delivery); err != nil {
			return err
		}
	}

	return nil
}

func (s *DeliveryService) ReportLocation(
	stream deliverypb.DeliveryService_ReportLocationServer,
) error {
	var deliveryID int64
	var pointsReceived int32

	for {
		point, err := stream.Recv()

		if err == io.EOF {
			if pointsReceived == 0 {
				return status.Error(
					codes.InvalidArgument,
					"no location points received",
				)
			}

			return stream.SendAndClose(&deliverypb.LocationReport{
				DeliveryId:     deliveryID,
				PointsReceived: pointsReceived,
			})
		}

		if err != nil {
			return err
		}

		id := point.GetDeliveryId()
		if id <= 0 {
			return status.Error(
				codes.InvalidArgument,
				"invalid delivery id",
			)
		}

		if pointsReceived == 0 {
			_, err := s.repository.Get(stream.Context(), id)
			if err != nil {
				return repositoryError(err)
			}

			deliveryID = id
		} else if id != deliveryID {
			return status.Error(
				codes.InvalidArgument,
				"all points must belong to one delivery",
			)
		}

		pointsReceived++
	}
}

func (s *DeliveryService) TrackDelivery(
	stream deliverypb.DeliveryService_TrackDeliveryServer,
) error {
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
			return status.Error(
				codes.InvalidArgument,
				"invalid delivery id",
			)
		}

		if acceptedPoints == 0 {
			_, err := s.repository.Get(stream.Context(), id)
			if err != nil {
				return repositoryError(err)
			}

			deliveryID = id
		} else if id != deliveryID {
			return status.Error(
				codes.InvalidArgument,
				"delivery id mismatch",
			)
		}

		acceptedPoints++

		err = stream.Send(&deliverypb.LocationAck{
			DeliveryId:     deliveryID,
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

func repositoryError(err error) error {
	if errors.Is(err, ErrDeliveryNotFound) {
		return status.Error(
			codes.NotFound,
			"delivery not found",
		)
	}

	return status.Error(
		codes.Internal,
		"internal server error",
	)
}
