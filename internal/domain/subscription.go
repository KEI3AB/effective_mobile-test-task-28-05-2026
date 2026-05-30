package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ServiceName string
	Price       int
	StartDate   time.Time
	EndDate     *time.Time
}
