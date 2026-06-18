package service

import (
	"testing"
	"time"

	"github/t-takamichi/fintech-game/backend/bank/internal/entity"

	"github.com/google/uuid"
)

func TestMapper(t *testing.T) {
	userID := uuid.New()
	idempKey := uuid.New()
	now := time.Now()

	entTx := entity.Transaction{
		ID:             123,
		UserID:         userID,
		Type:           "BUY",
		Amount:         -500,
		BalanceAfter:   1500,
		Description:    "テスト",
		IsPrinted:      true,
		IdempotencyKey: &idempKey,
		CreatedAt:      now,
	}

	domTx := toDomainTransaction(entTx)
	if domTx.ID != 123 || domTx.UserID != userID || domTx.Type != "BUY" || domTx.Amount != -500 || domTx.BalanceAfter != 1500 || domTx.Description != "テスト" || !domTx.IsPrinted || *domTx.IdempotencyKey != idempKey || !domTx.CreatedAt.Equal(now) {
		t.Errorf("failed to map transaction: %+v", domTx)
	}

	domTxs := toDomainTransactions([]entity.Transaction{entTx})
	if len(domTxs) != 1 || domTxs[0].ID != 123 {
		t.Errorf("failed to map transaction list")
	}

	master := &entity.AccountMaster{
		UserID:      userID,
		SubjectID:   "subj-1",
		CreditScore: 7,
		IsFrozen:    false,
		CurrentTurn: 2,
		AccountBalance: &entity.AccountBalance{
			UserID:        userID,
			Balance:       2000,
			LoanPrincipal: 500,
		},
	}

	domAcc := toDomainAccount(master)
	if domAcc.UserID != userID || domAcc.Balance != 2000 || domAcc.LoanPrincipal != 500 || domAcc.NetAsset != 1500 || domAcc.IsDebt != true || domAcc.IsFrozen || domAcc.CurrentTurn != 2 || domAcc.CreditScore != 7 {
		t.Errorf("failed to map account: %+v", domAcc)
	}

	domStatus := toDomainAccountStatus(master)
	if domStatus.UserID != userID || domStatus.Balance != 2000 || domStatus.LoanPrincipal != 500 || domStatus.NetAsset != 1500 || domStatus.IsDebt != true || domStatus.IsFrozen || domStatus.CurrentTurn != 2 || domStatus.CreditScore != 7 {
		t.Errorf("failed to map account status: %+v", domStatus)
	}
}
