package service

import (
	"context"
	"errors"

	deliverypb "github.com/slanalan1203/courier-flow/api"
)

var ErrDeliveryNotFound = errors.New("delivery not found")

type DeliveryRepository interface {
	Create(
		ctx context.Context,
		address string,
		status deliverypb.DeliveryStatus) (*deliverypb.Delivery, error)

	Get(
		ctx context.Context,
		id int64) (*deliverypb.Delivery, error)

	UpdateStatus(
		ctx context.Context,
		id int64, status deliverypb.DeliveryStatus) (*deliverypb.Delivery, error)

	List(
		ctx context.Context,
	) ([]*deliverypb.Delivery, error)
}
