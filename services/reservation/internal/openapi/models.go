package openapi

import (
	"github.com/google/uuid"
	"time"
)

type reservation struct {
	ID             int       `db:"id"`
	BookUid        uuid.UUID `db:"book_uid"`
	LibraryUid     uuid.UUID `db:"library_uid"`
	ReservationUid uuid.UUID `db:"reservation_uid"`
	StartDate      time.Time `db:"start_date"`
	Status         string    `db:"status"`
	TillDate       time.Time `db:"till_date"`
	Username       string    `db:"username"`
}

const (
	rented   = "RENTED"
	returned = "RETURNED"
	expired  = "EXPIRED"
)
