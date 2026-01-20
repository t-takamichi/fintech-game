package service

import (
	"context"
	"errors"
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	"github/t-takamichi/fintech-game/backend/bank/internal/entity"
	repository "github/t-takamichi/fintech-game/backend/bank/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DebtThreshold は口座が債務（負債）と見なされる閾値です。
const DebtThreshold int64 = 0

type AccountService interface {
	GetAccountStatus(subjectID string) (domain.AccountStatus, error)
	CreateAccount(ctx context.Context, subjectID string, initialScore int) (domain.Account, error)
}

type accountService struct {
	accountRepository        repository.AccountRepository
	accountBalanceRepository repository.AccountBalanceRepository
	db                       *gorm.DB
}

func NewAccountService(r repository.AccountRepository, b repository.AccountBalanceRepository, db *gorm.DB) AccountService {
	return &accountService{accountRepository: r, accountBalanceRepository: b, db: db}
}

func (s *accountService) GetAccountStatus(subjectID string) (domain.AccountStatus, error) {
	m, err := s.accountRepository.GetMasterByID(subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AccountStatus{}, errors.New("account not found")
		}
		return domain.AccountStatus{}, err
	}
	if m == nil {
		// mがnilの場合の処理ってそもそもNotFoundエラーにじゃあないよ。だってErrRecordNotFoundは上でキャッチしてるし,どちらかというとシステムエラーじゃないの？
		return domain.AccountStatus{}, errors.New("account not found")
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

func (s *accountService) CreateAccount(ctx context.Context, subjectID string, initialScore int) (domain.Account, error) {

	// FIXME: バリデーションを追加する
	_, verr := s.accountRepository.GetMasterByID(subjectID)

	if verr == nil {
		return domain.Account{}, errors.New("account already exists")
	}

	// FIXME: UUID生成を外部から注入できるようにする
	id := uuid.New()
	master := &entity.AccountMaster{UserID: id, SubjectID: subjectID, CreditScore: initialScore, IsFrozen: false, CurrentTurn: 0}
	balance := &entity.AccountBalance{UserID: id, Balance: 0, LoanPrincipal: 0}

	var created *entity.AccountMaster
	// FIXME: エラーハンドリングを改善する。
	// FIXME: トランザクションの扱いをRepository層に移すべきか検討する。
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.accountRepository.CreateMasterTx(ctx, tx, master); err != nil {
		}
		if _, err := s.accountBalanceRepository.CreateAccountBalanceTx(ctx, tx, balance); err != nil {
			return err
		}

		if err := tx.Preload("AccountBalance").First(&created, "user_id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}

	return domain.Account{
		UserID:        created.UserID,
		Balance:       created.AccountBalance.Balance,
		LoanPrincipal: created.AccountBalance.LoanPrincipal,
		NetAsset:      created.AccountBalance.Balance - created.AccountBalance.LoanPrincipal,
		IsDebt:        created.AccountBalance.LoanPrincipal > 0,
		IsFrozen:      created.IsFrozen,
		CurrentTurn:   created.CurrentTurn,
		CreditScore:   created.CreditScore,
	}, nil
}
