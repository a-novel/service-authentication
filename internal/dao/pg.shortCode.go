package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShortCode struct {
	bun.BaseModel `bun:"table:short_codes"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	Code string `bun:"code"`

	Usage  string `bun:"usage"`
	Target string `bun:"target"`

	Data []byte `bun:"data"`

	CreatedAt time.Time `bun:"created_at"`
	ExpiresAt time.Time `bun:"expires_at"`

	DeletedAt      *time.Time `bun:"deleted_at"`
	DeletedComment *string    `bun:"deleted_comment"`
}
