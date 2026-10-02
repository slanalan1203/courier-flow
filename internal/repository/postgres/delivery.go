package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	deliverypb "github.com/slanalan1203/courier-flow/api"
	deliveryservice "github.com/slanalan1203/courier-flow/internal/service"
)

type DeliveryRepository struct {
	pool *pgxpool.Pool
}

func (d *DeliveryRepository) Create(ctx context.Context, address string, status deliverypb.DeliveryStatus) (*deliverypb.Delivery, error) {
	const query = `
		INSERT INTO deliveries (address, status)
		VALUES ($1, $2)
		RETURNING id, address, status
	`
	delivery := &deliverypb.Delivery{}
	var statusValue int32

	err := d.pool.QueryRow(
		ctx,
		query,
		address,
		int32(status),
	).Scan(
		&delivery.Id,
		&delivery.Address,
		&statusValue,
	)
	if err != nil {
		return nil, fmt.Errorf("create delivery:%w", err)
	}
	delivery.Status = deliverypb.DeliveryStatus(statusValue)
	return delivery, nil

}

func (d *DeliveryRepository) Get(ctx context.Context, id int64) (*deliverypb.Delivery, error) {
	const query = `
SELECT id, address, status
FROM deliveries
WHERE id = $1`
	delivery := &deliverypb.Delivery{}
	var statusValue int32
	err := d.pool.QueryRow(
		ctx, query, id).Scan(
		&delivery.Id,
		&delivery.Address,
		&statusValue,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, deliveryservice.ErrDeliveryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get delivery:%w", err)
	}
	delivery.Status = deliverypb.DeliveryStatus(statusValue)
	return delivery, nil
}

func (d *DeliveryRepository) UpdateStatus(ctx context.Context, id int64, status deliverypb.DeliveryStatus) (*deliverypb.Delivery, error) {
	const query = `
UPDATE deliveries
SET status = $2
WHERE id = $1
RETURNING id, address, status`
	delivery := &deliverypb.Delivery{}
	var statusValue int32

	err := d.pool.QueryRow(ctx, query, id, int32(status)).Scan(
		&delivery.Id, &delivery.Address, &statusValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, deliveryservice.ErrDeliveryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update delivery:%w", err)
	}
	delivery.Status = deliverypb.DeliveryStatus(statusValue)
	return delivery, nil
}

func (d *DeliveryRepository) List(ctx context.Context) ([]*deliverypb.Delivery, error) {
	const query = `
SELECT id, address, status
FROM deliveries
ORDER BY id`
	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list delivery:%w", err)
	}
	defer rows.Close()
	deliveries := make([]*deliverypb.Delivery, 0)
	for rows.Next() {
		delivery := &deliverypb.Delivery{}
		var statusValue int32
		err := rows.Scan(
			&delivery.Id,
			&delivery.Address,
			&statusValue)
		if err != nil {
			return nil, fmt.Errorf("scan delivery:%w", err)
		}
		delivery.Status = deliverypb.DeliveryStatus(statusValue)
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list delivery:%w", err)
	}
	return deliveries, nil
}

func NewDeliveryRepository(pool *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{
		pool: pool,
	}
}
