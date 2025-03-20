package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Email     string
	Role      CredentialsRole
	CreatedAt time.Time
	UpdatedAt time.Time
}
