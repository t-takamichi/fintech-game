package entity

import (
	"time"

	"github.com/google/uuid"
)

type AccountBalance struct {
	UserID        uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	Balance       int64     `gorm:"not null;column:balance"`
	LoanPrincipal int64     `gorm:"not null;column:loan_principal"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (AccountBalance) TableName() string {
	return "accounts_balance"
}
