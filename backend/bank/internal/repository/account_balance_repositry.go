package repository

import (
	"context"
	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"gorm.io/gorm"
)

type AccountBalanceRepository interface {
	CreateAccountBalanceTx(ctx context.Context, tx *gorm.DB, ab entity.AccountBalance) (entity.AccountBalance, error)
}

type gormAccountBalanceRepository struct {
	db *gorm.DB
}

func NewAccountBalanceRepository(db *gorm.DB) AccountBalanceRepository {
	return &gormAccountBalanceRepository{db: db}
}

func (r *gormAccountBalanceRepository) CreateAccountBalanceTx(ctx context.Context, tx *gorm.DB, ab entity.AccountBalance) (entity.AccountBalance, error) {

	if err := tx.WithContext(ctx).Create(&ab).Error; err != nil {
		return entity.AccountBalance{}, err
	}
	return ab, nil
}
