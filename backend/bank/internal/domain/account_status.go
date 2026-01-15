package domain

import "github.com/google/uuid"

// AccountStatus is a response DTO for account status endpoints.
type AccountStatus struct {
	UserID        uuid.UUID `json:"user_id"`
	Balance       int64     `json:"balance"`
	LoanPrincipal int64     `json:"loan_principal"`
	NetAsset      int64     `json:"net_asset"`
	IsDebt        bool      `json:"is_debt"`
	IsFrozen      bool      `json:"is_frozen"`
	CurrentTurn   int       `json:"current_turn"`
	CreditScore   int       `json:"credit_score"`
}
