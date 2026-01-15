package repository

import (
	"github.com/google/uuid"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"gorm.io/gorm"
)

// AccountRepository defines DB operations for accounts (master/balance split).
type AccountRepository interface {
	GetMasterByID(userID string) (entity.AccountMaster, error)
}

type gormAccountRepo struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &gormAccountRepo{db: db}
}

func (r *gormAccountRepo) GetMasterByID(userID string) (entity.AccountMaster, error) {
	var m entity.AccountMaster
	uid, err := uuid.Parse(userID)
	if err != nil {
		return entity.AccountMaster{}, err
	}
	if err := r.db.Preload("AccountBalance").First(&m, "user_id = ?", uid).Error; err != nil {
		return entity.AccountMaster{}, err
	}
	return m, nil
}
