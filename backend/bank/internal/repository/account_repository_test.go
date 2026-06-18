package repository

import (
	"context"
	"testing"

	"github/t-takamichi/fintech-game/backend/bank/internal/db"
	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.InitDBFromEnv()
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	if err := gdb.Exec("TRUNCATE TABLE transactions, accounts_balance, accounts_master CASCADE").Error; err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	return gdb
}

func TestAccountRepository(t *testing.T) {
	gdb := setupTestDB(t)
	repo := NewAccountRepository(gdb)
	ctx := context.Background()

	userID := uuid.New()
	master := &entity.AccountMaster{
		UserID:      userID,
		SubjectID:   "test-subj",
		CreditScore: 5,
		IsFrozen:    false,
		CurrentTurn: 1,
	}

	err := gdb.Transaction(func(tx *gorm.DB) error {
		created, err := repo.CreateMasterTx(ctx, tx, master)
		if err != nil {
			return err
		}
		if created.UserID != userID {
			t.Errorf("expected UserID %v, got %v", userID, created.UserID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error on CreateMasterTx, got %v", err)
	}

	found, err := repo.GetMasterByID(ctx, "test-subj")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if found.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, found.UserID)
	}

	master.CreditScore = 8
	err = gdb.Transaction(func(tx *gorm.DB) error {
		updated, err := repo.UpdateMasterTx(ctx, tx, master)
		if err != nil {
			return err
		}
		if updated.CreditScore != 8 {
			t.Errorf("expected updated score 8, got %d", updated.CreditScore)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error on UpdateMasterTx, got %v", err)
	}

	userID2 := uuid.New()
	master2 := &entity.AccountMaster{
		UserID:      userID2,
		SubjectID:   "test-subj-2",
		CreditScore: 3,
		IsFrozen:    false,
		CurrentTurn: 1,
	}
	err = gdb.Transaction(func(tx *gorm.DB) error {
		_, err := repo.CreateMasterTx(ctx, tx, master2)
		return err
	})
	if err != nil {
		t.Fatalf("failed to create second master: %v", err)
	}

	page, err := repo.GetMastersPage(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get masters page: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("expected 2 masters, got %d", len(page))
	}

	var firstUserID, secondUserID uuid.UUID
	if userID.String() < userID2.String() {
		firstUserID = userID
		secondUserID = userID2
	} else {
		firstUserID = userID2
		secondUserID = userID
	}
	if page[0].UserID != firstUserID || page[1].UserID != secondUserID {
		t.Errorf("page elements are not ordered by user_id ASC")
	}
}
