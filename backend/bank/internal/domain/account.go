package domain

import "github.com/google/uuid"

type Account struct {
	UserID        uuid.UUID
	Balance       int64
	LoanPrincipal int64
	NetAsset      int64
	IsDebt        bool
	IsFrozen      bool
	CurrentTurn   int
	CreditScore   int
}
