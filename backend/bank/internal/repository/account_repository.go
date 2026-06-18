package repository

import (
	"context"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccountRepository interface {
	GetMasterByID(ctx context.Context, subjectID string) (*entity.AccountMaster, error)
	GetMasterForUpdateTx(ctx context.Context, tx *gorm.DB, subjectID string) (*entity.AccountMaster, error)
	CreateMasterTx(ctx context.Context, tx *gorm.DB, am *entity.AccountMaster) (*entity.AccountMaster, error)
	UpdateMasterTx(ctx context.Context, tx *gorm.DB, am *entity.AccountMaster) (*entity.AccountMaster, error)
	GetAllMasters(ctx context.Context) ([]entity.AccountMaster, error)
	GetMastersPage(ctx context.Context, limit, offset int) ([]entity.AccountMaster, error)
}

type gormAccountRepo struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &gormAccountRepo{db: db}
}

func (r *gormAccountRepo) GetMasterByID(ctx context.Context, subjectID string) (*entity.AccountMaster, error) {
	var m entity.AccountMaster
	if err := r.db.WithContext(ctx).Preload("AccountBalance").First(&m, "subject_id = ?", subjectID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormAccountRepo) GetMasterForUpdateTx(ctx context.Context, tx *gorm.DB, subjectID string) (*entity.AccountMaster, error) {
	var m entity.AccountMaster
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Preload("AccountBalance").First(&m, "subject_id = ?", subjectID).Error; err != nil {
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

func (r *gormAccountRepo) UpdateMasterTx(ctx context.Context, tx *gorm.DB, am *entity.AccountMaster) (*entity.AccountMaster, error) {
	if err := tx.WithContext(ctx).Save(am).Error; err != nil {
		return nil, err
	}
	return am, nil
}

func (r *gormAccountRepo) GetAllMasters(ctx context.Context) ([]entity.AccountMaster, error) {
	var list []entity.AccountMaster
	if err := r.db.WithContext(ctx).Preload("AccountBalance").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *gormAccountRepo) GetMastersPage(ctx context.Context, limit, offset int) ([]entity.AccountMaster, error) {
	var list []entity.AccountMaster
	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Preload("AccountBalance").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
