package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Credentials struct {
	bun.BaseModel `bun:"table:credentials"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	Email    string `bun:"email"`
	Password string `bun:"password"`

	Role string `bun:"role"`

	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}
