package domain

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID           int64     `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Description  string    `json:"description"`
	IsPrinted      bool       `json:"is_printed"`
	IdempotencyKey *uuid.UUID `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
