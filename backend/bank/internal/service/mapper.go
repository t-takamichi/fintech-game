package service

import (
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	"github/t-takamichi/fintech-game/backend/bank/internal/entity"
)

func toDomainTransaction(e entity.Transaction) domain.Transaction {
	return domain.Transaction{
		ID:           e.ID,
		UserID:       e.UserID,
		Type:         e.Type,
		Amount:       e.Amount,
		BalanceAfter: e.BalanceAfter,
		Description:  e.Description,
		IsPrinted:    e.IsPrinted,
		IdempotencyKey: e.IdempotencyKey,
		CreatedAt:    e.CreatedAt,
	}
}

func toDomainTransactions(entities []entity.Transaction) []domain.Transaction {
	list := make([]domain.Transaction, len(entities))
	for i, e := range entities {
		list[i] = toDomainTransaction(e)
	}
	return list
}

func toDomainAccount(created *entity.AccountMaster) domain.Account {
	return domain.Account{
		UserID:        created.UserID,
		Balance:       created.AccountBalance.Balance,
		LoanPrincipal: created.AccountBalance.LoanPrincipal,
		NetAsset:      created.AccountBalance.Balance - created.AccountBalance.LoanPrincipal,
		IsDebt:        created.AccountBalance.LoanPrincipal > 0,
		IsFrozen:      created.IsFrozen,
		CurrentTurn:   created.CurrentTurn,
		CreditScore:   created.CreditScore,
	}
}
