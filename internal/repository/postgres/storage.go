package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/effective_mobile-test-task-28-05-2026/internal/domain"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionStorage struct {
	pool *pgxpool.Pool
}

func NewSubscriptionStorage(pool *pgxpool.Pool) *SubscriptionStorage {
	return &SubscriptionStorage{
		pool: pool,
	}
}

var (
	ErrNoRowsAffected = errors.New("no rows affected")
)

type Subscription struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`
	ServiceName string     `db:"service_name"`
	Price       int        `db:"price"`
	StartDate   time.Time  `db:"start_date"`
	EndDate     *time.Time `db:"end_date"`
}

func (r *SubscriptionStorage) Create(ctx context.Context, sub domain.Subscription) (uuid.UUID, error) {
	query := `
		INSERT INTO "subscription" (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (r *SubscriptionStorage) Read(ctx context.Context, subID uuid.UUID) (domain.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM "subscription"
		WHERE id = $1;
	`

	rows, err := r.pool.Query(ctx, query, subID)
	if err != nil {
		return domain.Subscription{}, err
	}

	dbSub, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Subscription])
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Subscription{}, ErrNoRowsAffected
		}
		return domain.Subscription{}, err
	}

	return r.mapDBToDomainSubscription(dbSub), nil
}

func (r *SubscriptionStorage) Update(ctx context.Context, sub domain.Subscription) error {
	query := `
		UPDATE "subscription"
		SET service_name = $2, price = $3, user_id = $4, start_date = $5, end_date = $6
		WHERE id = $1;
	`

	tag, err := r.pool.Exec(ctx, query,
		sub.ID,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

func (r *SubscriptionStorage) Delete(ctx context.Context, subID uuid.UUID) error {
	query := `
		DELETE FROM "subscription"
		WHERE id = $1;
	`

	tag, err := r.pool.Exec(ctx, query, subID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

func (r *SubscriptionStorage) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM "subscription"
		WHERE user_id = $1
		ORDER BY start_date ASC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	dbSubs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Subscription])
	if err != nil {
		return nil, err
	}

	subs := make([]domain.Subscription, 0, len(dbSubs))
	for _, sub := range dbSubs {
		subs = append(subs, r.mapDBToDomainSubscription(sub))
	}

	return subs, nil
}

func (r *SubscriptionStorage) GetFromPeriod(ctx context.Context, userID uuid.UUID, serviceName string, periodStart time.Time, periodEnd *time.Time) ([]domain.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
        FROM "subscription"
        WHERE user_id = $1
		AND service_name = $2
		AND (end_date IS NULL OR end_date >= $3)
		AND ($4::date IS NULL OR start_date <= $4);
	`

	rows, err := r.pool.Query(ctx, query, userID, serviceName, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	dbSubs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Subscription])
	if err != nil {
		return nil, err
	}

	subs := make([]domain.Subscription, 0, len(dbSubs))
	for _, sub := range dbSubs {
		subs = append(subs, r.mapDBToDomainSubscription(sub))
	}

	return subs, nil
}

func (r *SubscriptionStorage) mapDBToDomainSubscription(sub Subscription) domain.Subscription {
	return domain.Subscription{
		ID:          sub.ID,
		UserID:      sub.UserID,
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		StartDate:   sub.StartDate,
		EndDate:     sub.EndDate,
	}
}
