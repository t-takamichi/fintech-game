package service

import (
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	repository "github/t-takamichi/fintech-game/backend/bank/internal/repository"
)

// DebtThreshold は口座が債務（負債）と見なされる閾値です。
const DebtThreshold int64 = 0

type AccountService interface {
	GetAccountStatus(subjectID string) (domain.AccountStatus, error)
}

type accountService struct {
	repo repository.AccountRepository
}

func NewAccountService(r repository.AccountRepository) AccountService {
	return &accountService{repo: r}
}

func (s *accountService) GetAccountStatus(subjectID string) (domain.AccountStatus, error) {
	m, err := s.repo.GetMasterByID(subjectID)
	if err != nil {
		return domain.AccountStatus{}, err
	}

	netAsset := m.AccountBalance.Balance - m.AccountBalance.LoanPrincipal
	isDebt := m.AccountBalance.LoanPrincipal > 0

	return domain.AccountStatus{
		UserID:        m.UserID,
		Balance:       m.AccountBalance.Balance,
		LoanPrincipal: m.AccountBalance.LoanPrincipal,
		NetAsset:      netAsset,
		IsDebt:        isDebt,
		IsFrozen:      m.IsFrozen,
		CurrentTurn:   m.CurrentTurn,
		CreditScore:   m.CreditScore,
	}, nil
}
