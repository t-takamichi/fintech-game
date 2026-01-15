package entity

import (
	"time"

	"github.com/google/uuid"
)

type AccountMaster struct {
	UserID         uuid.UUID      `gorm:"type:uuid;primaryKey;column:user_id"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	SubjectID      string         `gorm:"type:text;column:subject_id;index"`
	CreditScore    int            `gorm:"not null;column:credit_score"`
	IsFrozen       bool           `gorm:"not null;column:is_frozen"`
	CurrentTurn    int            `gorm:"not null;column:current_turn"`
	AccountBalance AccountBalance `gorm:"foreignKey:UserID;references:UserID"`
}

func (AccountMaster) TableName() string {
	return "accounts_master"
}
