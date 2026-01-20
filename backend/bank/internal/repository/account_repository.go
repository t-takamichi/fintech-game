package repository

import (
	"context"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"gorm.io/gorm"
)

type AccountRepository interface {
	GetMasterByID(subjectID string) (*entity.AccountMaster, error)
	CreateMasterTx(ctx context.Context, tx *gorm.DB, am *entity.AccountMaster) (*entity.AccountMaster, error)
}

type gormAccountRepo struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &gormAccountRepo{db: db}
}

func (r *gormAccountRepo) GetMasterByID(subjectID string) (*entity.AccountMaster, error) {
	var m entity.AccountMaster
	if err := r.db.Preload("AccountBalance").First(&m, "subject_id = ?", subjectID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormAccountRepo) CreateMasterTx(ctx context.Context, tx *gorm.DB, am *entity.AccountMaster) (*entity.AccountMaster, error) {
	err := tx.WithContext(ctx).Create(am).Error
	if err != nil {
		return nil, err
	}
	return am, nil
}
