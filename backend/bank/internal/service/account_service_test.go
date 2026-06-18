package service

import (
	"context"
	"testing"

	"github/t-takamichi/fintech-game/backend/bank/internal/db"
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	repository "github/t-takamichi/fintech-game/backend/bank/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, AccountService) {
	t.Helper()
	gdb, err := db.InitDBFromEnv()
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	if err := gdb.Exec("TRUNCATE TABLE transactions, accounts_balance, accounts_master CASCADE").Error; err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	accountRepo := repository.NewAccountRepository(gdb)
	balanceRepo := repository.NewAccountBalanceRepository(gdb)
	txRepo := repository.NewTransactionRepository(gdb)

	svc := NewAccountService(accountRepo, balanceRepo, txRepo, gdb)
	return gdb, svc
}

func TestCreateAccount(t *testing.T) {
	_, svc := setupTestDB(t)
	ctx := context.Background()

	subjectID := "test-user-1"
	acc, berr := svc.CreateAccount(ctx, subjectID, 3)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}
	if acc.CreditScore != 3 {
		t.Errorf("expected score 3, got %d", acc.CreditScore)
	}
	if acc.IsFrozen {
		t.Errorf("expected account to be active, but frozen")
	}

	_, berr = svc.CreateAccount(ctx, subjectID, 5)
	if berr == nil || berr.Type != domain.ErrorTypeAlreadyExists {
		t.Fatalf("expected AlreadyExists error, got %v", berr)
	}

	_, berr = svc.CreateAccount(ctx, "test-user-2", 11)
	if berr == nil || berr.Type != domain.ErrorTypeInconsistent {
		t.Fatalf("expected validation error, got %v", berr)
	}
}

func TestInitializeAccount(t *testing.T) {
	_, svc := setupTestDB(t)
	ctx := context.Background()

	subjectID := "test-user-1"
	_, berr := svc.CreateAccount(ctx, subjectID, 3)
	if berr != nil {
		t.Fatalf("failed to create account: %v", berr)
	}

	idempKey := uuid.New()
	acc, berr := svc.InitializeAccount(ctx, subjectID, &idempKey)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}
	if acc.Balance != 1000000 {
		t.Errorf("expected balance 1,000,000, got %d", acc.Balance)
	}
	if acc.LoanPrincipal != 1000000 {
		t.Errorf("expected loan principal 1,000,000, got %d", acc.LoanPrincipal)
	}

	acc2, berr := svc.InitializeAccount(ctx, subjectID, &idempKey)
	if berr != nil {
		t.Fatalf("expected nil error on duplicate request, got %v", berr)
	}
	if acc2.Balance != acc.Balance {
		t.Errorf("expected same balance, got %d", acc2.Balance)
	}

	idempKey2 := uuid.New()
	_, berr = svc.InitializeAccount(ctx, subjectID, &idempKey2)
	if berr == nil || berr.Type != domain.ErrorTypeAlreadyExists {
		t.Fatalf("expected AlreadyExists error on subsequent initialize, got %v", berr)
	}
}

func TestExecuteTransaction(t *testing.T) {
	_, svc := setupTestDB(t)
	ctx := context.Background()

	subjectID := "test-user-1"
	_, berr := svc.CreateAccount(ctx, subjectID, 3)
	if berr != nil {
		t.Fatalf("failed to create account: %v", berr)
	}

	idempInit := uuid.New()
	_, berr = svc.InitializeAccount(ctx, subjectID, &idempInit)
	if berr != nil {
		t.Fatalf("failed to initialize account: %v", berr)
	}

	idempTx1 := uuid.New()
	acc, berr := svc.ExecuteTransaction(ctx, subjectID, -200000, "BUY", "ロケット株購入", &idempTx1)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}
	if acc.Balance != 800000 {
		t.Errorf("expected balance 800,000, got %d", acc.Balance)
	}

	acc2, berr := svc.ExecuteTransaction(ctx, subjectID, -200000, "BUY", "ロケット株購入", &idempTx1)
	if berr != nil {
		t.Fatalf("expected nil error on duplicate transaction, got %v", berr)
	}
	if acc2.Balance != acc.Balance {
		t.Errorf("expected same balance, got %d", acc2.Balance)
	}

	idempTx2 := uuid.New()
	acc3, berr := svc.ExecuteTransaction(ctx, subjectID, -100000, "REPAYMENT", "ローン返済", &idempTx2)
	if berr != nil {
		t.Fatalf("expected nil error on repayment, got %v", berr)
	}
	if acc3.Balance != 700000 {
		t.Errorf("expected balance 700,000 after repayment, got %d", acc3.Balance)
	}
	if acc3.LoanPrincipal != 900000 {
		t.Errorf("expected loan principal 900,000 after repayment, got %d", acc3.LoanPrincipal)
	}

	idempTx3 := uuid.New()
	acc4, berr := svc.ExecuteTransaction(ctx, subjectID, -1000000, "REPAYMENT", "過剰ローン返済", &idempTx3)
	if berr != nil {
		t.Fatalf("expected nil error on excess repayment, got %v", berr)
	}
	if acc4.Balance != -200000 {
		t.Errorf("expected balance -200,000, got %d", acc4.Balance)
	}
	if acc4.LoanPrincipal != 0 {
		t.Errorf("expected loan principal 0, got %d", acc4.LoanPrincipal)
	}
}

func TestSettleAccount(t *testing.T) {
	_, svc := setupTestDB(t)
	ctx := context.Background()

	subjectID := "test-user-1"
	_, berr := svc.CreateAccount(ctx, subjectID, 3)
	if berr != nil {
		t.Fatalf("failed to create account: %v", berr)
	}

	acc, berr := svc.SettleAccount(ctx, subjectID)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}
	if acc.CreditScore != 4 {
		t.Errorf("expected score 4, got %d", acc.CreditScore)
	}

	acc2, berr := svc.SettleAccount(ctx, subjectID)
	if berr != nil {
		t.Fatalf("expected nil error on duplicate settle, got %v", berr)
	}
	if acc2.CreditScore != 4 {
		t.Errorf("expected score to remain 4 due to idempotency, got %d", acc2.CreditScore)
	}
}

func TestApplyInterestBatch(t *testing.T) {
	_, svc := setupTestDB(t)
	ctx := context.Background()

	users := []string{"user-a", "user-b"}
	for _, u := range users {
		_, berr := svc.CreateAccount(ctx, u, 3)
		if berr != nil {
			t.Fatalf("failed to create account: %v", berr)
		}
		idemp := uuid.New()
		_, berr = svc.InitializeAccount(ctx, u, &idemp)
		if berr != nil {
			t.Fatalf("failed to initialize: %v", berr)
		}
	}

	berr := svc.ApplyInterestBatch(ctx)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}

	for _, u := range users {
		status, berr := svc.GetAccountStatus(ctx, u)
		if berr != nil {
			t.Fatalf("failed to get status: %v", berr)
		}
		if status.LoanPrincipal != 1100000 {
			t.Errorf("expected loan principal 1,100,000, got %d", status.LoanPrincipal)
		}
	}

	berr = svc.ApplyInterestBatch(ctx)
	if berr != nil {
		t.Fatalf("expected nil error on second run, got %v", berr)
	}
	for _, u := range users {
		status, berr := svc.GetAccountStatus(ctx, u)
		if berr != nil {
			t.Fatalf("failed to get status: %v", berr)
		}
		if status.LoanPrincipal != 1100000 {
			t.Errorf("expected loan principal to remain 1,100,000, got %d", status.LoanPrincipal)
		}
	}
}

func TestReconcileAccountsBatch(t *testing.T) {
	gdb, svc := setupTestDB(t)
	ctx := context.Background()

	subjectID := "test-user-1"
	acc, berr := svc.CreateAccount(ctx, subjectID, 3)
	if berr != nil {
		t.Fatalf("failed to create account: %v", berr)
	}

	inconsistent, berr := svc.ReconcileAccountsBatch(ctx)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}
	if len(inconsistent) != 0 {
		t.Errorf("expected 0 inconsistent accounts, got %d", len(inconsistent))
	}

	if err := gdb.Exec("UPDATE accounts_balance SET balance = 500 WHERE user_id = ?", acc.UserID).Error; err != nil {
		t.Fatalf("failed to manually update balance: %v", err)
	}

	inconsistent, berr = svc.ReconcileAccountsBatch(ctx)
	if berr != nil {
		t.Fatalf("expected nil error, got %v", berr)
	}
	if len(inconsistent) != 1 || inconsistent[0] != acc.UserID {
		t.Errorf("expected test-user-1 to be inconsistent, got %v", inconsistent)
	}

	status, berr := svc.GetAccountStatus(ctx, subjectID)
	if berr != nil {
		t.Fatalf("failed to get status: %v", berr)
	}
	if !status.IsFrozen {
		t.Errorf("expected account to be frozen due to inconsistency")
	}
}
