package repository

import (
	"context"

	"github.com/google/uuid"
	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"gorm.io/gorm"
)

type TransactionRepository interface {
	CreateTransactionTx(ctx context.Context, tx *gorm.DB, t *entity.Transaction) (*entity.Transaction, error)
	GetTransactionsByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Transaction, error)
	MarkAsPrintedTx(ctx context.Context, tx *gorm.DB, ids []int64) error
	GetTransactionSumByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetTransactionByIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, key uuid.UUID) (*entity.Transaction, error)
}

type gormTransactionRepo struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &gormTransactionRepo{db: db}
}

func (r *gormTransactionRepo) CreateTransactionTx(ctx context.Context, tx *gorm.DB, t *entity.Transaction) (*entity.Transaction, error) {
	if err := tx.WithContext(ctx).Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (r *gormTransactionRepo) GetTransactionsByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Transaction, error) {
	var list []entity.Transaction
	if err := r.db.WithContext(ctx).Order("created_at asc, id asc").Find(&list, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *gormTransactionRepo) MarkAsPrintedTx(ctx context.Context, tx *gorm.DB, ids []int64) error {
	return tx.WithContext(ctx).Model(&entity.Transaction{}).Where("id IN ?", ids).Update("is_printed", true).Error
}

func (r *gormTransactionRepo) GetTransactionSumByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Model(&entity.Transaction{}).Where("user_id = ?", userID).Select("COALESCE(SUM(amount), 0)").Scan(&sum).Error
	return sum, err
}

func (r *gormTransactionRepo) GetTransactionByIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, key uuid.UUID) (*entity.Transaction, error) {
	var t entity.Transaction
	if err := tx.WithContext(ctx).First(&t, "idempotency_key = ?", key).Error; err != nil {
		return nil, err
	}
	return &t, nil
}
