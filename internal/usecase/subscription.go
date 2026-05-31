package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/effective_mobile-test-task-28-05-2026/internal/domain"
	"github.com/google/uuid"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub domain.Subscription) (uuid.UUID, error)
	Read(ctx context.Context, subID uuid.UUID) (domain.Subscription, error)
	Update(ctx context.Context, sub domain.Subscription) error
	Delete(ctx context.Context, subID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Subscription, error)
	GetFromPeriod(ctx context.Context, userID uuid.UUID, serviceName string, periodStart time.Time, periodEnd *time.Time) ([]domain.Subscription, error)
}

var (
	ErrNotFound         = errors.New("subscription not found")
	ErrInvalidDates     = errors.New("end date cannot be before start date")
	ErrInvalidPrice     = errors.New("price cannot be negative")
	ErrEmptyServiceName = errors.New("service name cannot be empty")
	ErrNegativeLimit    = errors.New("limit cannot be negative")
	ErrNegativeOffset   = errors.New("offset cannot be negative")
	ErrTooBigQuery      = errors.New("limit cannot be that big")
)

const (
	MaxLimit     = 100
	DefaultLimit = 20
	MinOffset    = 0
)

type SubscriptionUseCase struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

func NewSubscriptionUseCase(repo SubscriptionRepository, log *slog.Logger) *SubscriptionUseCase {
	return &SubscriptionUseCase{repo: repo, log: log}
}

func (uc *SubscriptionUseCase) Create(ctx context.Context, sub domain.Subscription) (uuid.UUID, error) {
	err := uc.validateSubscription(sub)
	if err != nil {
		return uuid.Nil, err
	}

	subID, err := uc.repo.Create(ctx, sub)
	if err != nil {
		return uuid.Nil, err
	}

	return subID, nil
}

func (uc *SubscriptionUseCase) Read(ctx context.Context, subID uuid.UUID) (domain.Subscription, error) {
	sub, err := uc.repo.Read(ctx, subID)
	if err != nil {
		return domain.Subscription{}, err
	}

	return sub, nil
}

func (uc *SubscriptionUseCase) Update(ctx context.Context, sub domain.Subscription) error {
	err := uc.validateSubscription(sub)
	if err != nil {
		return err
	}

	err = uc.repo.Update(ctx, sub)
	if err != nil {
		return err
	}

	return nil
}

func (uc *SubscriptionUseCase) Delete(ctx context.Context, subID uuid.UUID) error {
	err := uc.repo.Delete(ctx, subID)
	if err != nil {
		return err
	}

	return nil
}

func (uc *SubscriptionUseCase) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Subscription, error) {
	if limit < 0 {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = MinOffset
	}

	if limit > MaxLimit {
		limit = MaxLimit
	}

	subs, err := uc.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

func (uc *SubscriptionUseCase) CalculatePriceFromPeriod(
	ctx context.Context,
	userID uuid.UUID,
	serviceName string,
	periodStart time.Time,
	periodEnd *time.Time,
) (int, error) {
	if periodEnd != nil && periodEnd.Before(periodStart) {
		return 0, ErrInvalidDates
	}

	subs, err := uc.repo.GetFromPeriod(ctx, userID, serviceName, periodStart, periodEnd)
	if err != nil {
		return 0, err
	}

	var totalPrice int
	for _, sub := range subs {
		actualStart := sub.StartDate
		if periodStart.After(actualStart) {
			actualStart = periodStart
		}

		var actualEnd time.Time
		if sub.EndDate != nil && periodEnd != nil {
			if sub.EndDate.Before(*periodEnd) {
				actualEnd = *sub.EndDate
			} else {
				actualEnd = *periodEnd
			}
		} else if sub.EndDate != nil {
			actualEnd = *sub.EndDate
		} else if periodEnd != nil {
			actualEnd = *periodEnd
		} else {
			actualEnd = time.Now()
		}

		months := (actualEnd.Year()-actualStart.Year())*12 + int(actualEnd.Month()) - int(actualStart.Month()) + 1

		if months > 0 {
			totalPrice += sub.Price * months
		}
	}

	return totalPrice, nil
}

func (uc *SubscriptionUseCase) validateSubscription(sub domain.Subscription) error {
	if sub.EndDate != nil && sub.EndDate.Before(sub.StartDate) {
		return ErrInvalidDates
	}
	if sub.Price < 0 {
		return ErrInvalidPrice
	}
	if sub.ServiceName == "" {
		return ErrEmptyServiceName
	}

	return nil
}
