package models

import (
	"github.com/google/uuid"
	"time"
)

type Retry struct {
	LibraryUID uuid.UUID
	BookUUID   uuid.UUID
	Condition  string
	Violations int
	Username   string
	Time       time.Time
}
