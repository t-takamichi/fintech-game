package entity

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID       uuid.UUID `gorm:"type:uuid;index;column:user_id"`
	Type         string    `gorm:"type:text;column:type"`
	Amount       int64     `gorm:"column:amount"`
	BalanceAfter int64     `gorm:"column:balance_after"`
	Description  string    `gorm:"size:15;column:description"`
	IsPrinted    bool      `gorm:"column:is_printed;default:false"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (Transaction) TableName() string {
	return "transactions"
}
