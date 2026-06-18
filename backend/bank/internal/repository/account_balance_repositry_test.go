package repository

import (
	"context"
	"testing"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestAccountBalanceRepository(t *testing.T) {
	gdb := setupTestDB(t)
	repo := NewAccountBalanceRepository(gdb)
	accountRepo := NewAccountRepository(gdb)
	ctx := context.Background()

	userID := uuid.New()
	master := &entity.AccountMaster{
		UserID:      userID,
		SubjectID:   "test-user",
		CreditScore: 5,
		IsFrozen:    false,
		CurrentTurn: 1,
	}
	err := gdb.Transaction(func(tx *gorm.DB) error {
		_, err := accountRepo.CreateMasterTx(ctx, tx, master)
		return err
	})
	if err != nil {
		t.Fatalf("failed to setup master: %v", err)
	}

	balance := &entity.AccountBalance{
		UserID:        userID,
		Balance:       1000,
		LoanPrincipal: 500,
	}
	err = gdb.Transaction(func(tx *gorm.DB) error {
		created, err := repo.CreateAccountBalanceTx(ctx, tx, balance)
		if err != nil {
			return err
		}
		if created.Balance != 1000 {
			t.Errorf("expected balance 1000, got %d", created.Balance)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error on CreateAccountBalanceTx, got %v", err)
	}

	balance.Balance = 2000
	err = gdb.Transaction(func(tx *gorm.DB) error {
		updated, err := repo.UpdateAccountBalanceTx(ctx, tx, balance)
		if err != nil {
			return err
		}
		if updated.Balance != 2000 {
			t.Errorf("expected updated balance 2000, got %d", updated.Balance)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error on UpdateAccountBalanceTx, got %v", err)
	}
}
