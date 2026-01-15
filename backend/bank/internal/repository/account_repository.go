package repository

import (
	"context"

	"github.com/google/uuid"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"gorm.io/gorm"
)

// AccountRepository defines DB operations for accounts (master/balance split).
type AccountRepository interface {
	GetMasterByID(subjectID string) (entity.AccountMaster, error)
	CreateMasterTx(ctx context.Context, tx *gorm.DB, am entity.AccountMaster) (entity.AccountMaster, error)
}

type gormAccountRepo struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &gormAccountRepo{db: db}
}

func (r *gormAccountRepo) GetMasterByID(subjectID string) (entity.AccountMaster, error) {
	var m entity.AccountMaster
	if uid, err := uuid.Parse(subjectID); err == nil {
		if err := r.db.Preload("AccountBalance").First(&m, "user_id = ?", uid).Error; err == nil {
			return m, nil
		}
	}

	if err := r.db.Preload("AccountBalance").First(&m, "subject_id = ?", subjectID).Error; err != nil {
		return entity.AccountMaster{}, err
	}
	return m, nil
}

func (r *gormAccountRepo) CreateMasterTx(ctx context.Context, tx *gorm.DB, am entity.AccountMaster) (entity.AccountMaster, error) {
	if err := tx.WithContext(ctx).Create(&am).Error; err != nil {
		return entity.AccountMaster{}, err
	}
	return am, nil
}
