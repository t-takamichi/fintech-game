package repository

import (
	"context"
	"testing"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestTransactionRepository(t *testing.T) {
	gdb := setupTestDB(t)
	repo := NewTransactionRepository(gdb)
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

	tx1 := &entity.Transaction{
		UserID:       userID,
		Type:         "BUY",
		Amount:       -100,
		BalanceAfter: 900,
		Description:  "BUY-1",
		IsPrinted:    false,
	}
	err = gdb.Transaction(func(tx *gorm.DB) error {
		_, err := repo.CreateTransactionTx(ctx, tx, tx1)
		return err
	})
	if err != nil {
		t.Fatalf("expected nil error on CreateTransactionTx, got %v", err)
	}

	list, err := repo.GetTransactionsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(list) != 1 || list[0].Description != "BUY-1" {
		t.Errorf("expected 1 transaction with DESC BUY-1, got %v", list)
	}

	sum, err := repo.GetTransactionSumByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if sum != -100 {
		t.Errorf("expected sum -100, got %d", sum)
	}

	idempKey := uuid.New()
	tx2 := &entity.Transaction{
		UserID:         userID,
		Type:           "SELL",
		Amount:         200,
		BalanceAfter:   1100,
		Description:    "SELL-1",
		IsPrinted:      false,
		IdempotencyKey: &idempKey,
	}
	err = gdb.Transaction(func(tx *gorm.DB) error {
		_, err := repo.CreateTransactionTx(ctx, tx, tx2)
		return err
	})
	if err != nil {
		t.Fatalf("failed to create second tx: %v", err)
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		found, err := repo.GetTransactionByIdempotencyKeyTx(ctx, tx, idempKey)
		if err != nil {
			return err
		}
		if found == nil || found.Description != "SELL-1" {
			t.Errorf("expected transaction with DESC SELL-1, got %v", found)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to fetch by idempotency key: %v", err)
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		return repo.MarkAsPrintedTx(ctx, tx, []int64{tx1.ID})
	})
	if err != nil {
		t.Fatalf("expected nil error on MarkAsPrintedTx, got %v", err)
	}

	list2, err := repo.GetTransactionsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get transactions: %v", err)
	}
	if !list2[0].IsPrinted {
		t.Errorf("expected first transaction to be printed")
	}
}
