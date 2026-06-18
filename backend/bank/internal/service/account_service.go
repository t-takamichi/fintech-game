package service

import (
	"context"
	"errors"
	"strconv"

	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	"github/t-takamichi/fintech-game/backend/bank/internal/entity"
	repository "github/t-takamichi/fintech-game/backend/bank/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const DebtThreshold int64 = 0

type AccountService interface {
	GetAccountStatus(ctx context.Context, subjectID string) (domain.AccountStatus, *domain.BankAccountError)
	CreateAccount(ctx context.Context, subjectID string, initialScore int) (domain.Account, *domain.BankAccountError)
	InitializeAccount(ctx context.Context, subjectID string, idempotencyKey *uuid.UUID) (domain.Account, *domain.BankAccountError)
	GetAccountHistory(ctx context.Context, subjectID string) ([]domain.Transaction, *domain.BankAccountError)
	MarkAsPrinted(ctx context.Context, subjectID string, ids []int64) *domain.BankAccountError
	SettleAccount(ctx context.Context, subjectID string) (domain.Account, *domain.BankAccountError)
	ExecuteTransaction(ctx context.Context, subjectID string, amount int64, txType string, desc string, idempotencyKey *uuid.UUID) (domain.Account, *domain.BankAccountError)
	ApplyInterestBatch(ctx context.Context) *domain.BankAccountError
	ReconcileAccountsBatch(ctx context.Context) ([]uuid.UUID, *domain.BankAccountError)
}

type accountService struct {
	accountRepository        repository.AccountRepository
	accountBalanceRepository repository.AccountBalanceRepository
	transactionRepository    repository.TransactionRepository
	db                       *gorm.DB
	uuidGenerator            func() uuid.UUID
}

func NewAccountService(r repository.AccountRepository, b repository.AccountBalanceRepository, t repository.TransactionRepository, db *gorm.DB) AccountService {
	return &accountService{
		accountRepository:        r,
		accountBalanceRepository: b,
		transactionRepository:    t,
		db:                       db,
		uuidGenerator:            uuid.New,
	}
}

// withTx トランザクション処理とエラーハンドリングの共通ラッパー
func (s *accountService) withTx(ctx context.Context, fn func(tx *gorm.DB) error) *domain.BankAccountError {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
	if err != nil {
		var berr *domain.BankAccountError
		if errors.As(err, &berr) {
			return berr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NewBankAccountError(domain.ErrorTypeNotFound, "resource not found")
		}
		return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}
	return nil
}

// getMasterForUpdate ロック付きでマスタを安全に取得する共通ヘルパー
func (s *accountService) getMasterForUpdate(ctx context.Context, tx *gorm.DB, subjectID string) (*entity.AccountMaster, error) {
	master, err := s.accountRepository.GetMasterForUpdateTx(ctx, tx, subjectID)
	if err != nil {
		return nil, err
	}
	if master == nil {
		return nil, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
	}
	if master.AccountBalance == nil {
		return nil, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account balance is nil")
	}
	return master, nil
}

// checkIdempotency べき等キーの重複チェック
func (s *accountService) checkIdempotency(ctx context.Context, tx *gorm.DB, key *uuid.UUID) (bool, error) {
	if key == nil {
		return false, nil
	}
	exist, err := s.transactionRepository.GetTransactionByIdempotencyKeyTx(ctx, tx, *key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return exist != nil, nil
}

func (s *accountService) GetAccountStatus(ctx context.Context, subjectID string) (domain.AccountStatus, *domain.BankAccountError) {
	m, err := s.accountRepository.GetMasterByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}
	if m == nil || m.AccountBalance == nil {
		return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "invalid account state")
	}

	netAsset := m.AccountBalance.Balance - m.AccountBalance.LoanPrincipal
	return domain.AccountStatus{
		UserID:        m.UserID,
		Balance:       m.AccountBalance.Balance,
		LoanPrincipal: m.AccountBalance.LoanPrincipal,
		NetAsset:      netAsset,
		IsDebt:        m.AccountBalance.LoanPrincipal > 0,
		IsFrozen:      m.IsFrozen,
		CurrentTurn:   m.CurrentTurn,
		CreditScore:   m.CreditScore,
	}, nil
}

func (s *accountService) CreateAccount(ctx context.Context, subjectID string, initialScore int) (domain.Account, *domain.BankAccountError) {
	if subjectID == "" {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "subject_id is required")
	}
	if initialScore < 1 || initialScore > 10 {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "initial_score must be between 1 and 10")
	}

	_, verr := s.accountRepository.GetMasterByID(ctx, subjectID)
	if verr == nil {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeAlreadyExists, "account already exists")
	}

	id := s.uuidGenerator()
	master := &entity.AccountMaster{UserID: id, SubjectID: subjectID, CreditScore: initialScore, IsFrozen: false, CurrentTurn: 0}
	balance := &entity.AccountBalance{UserID: id, Balance: 0, LoanPrincipal: 0}

	err := s.withTx(ctx, func(tx *gorm.DB) error {
		if _, err := s.accountRepository.CreateMasterTx(ctx, tx, master); err != nil {
			return err
		}
		if _, err := s.accountBalanceRepository.CreateAccountBalanceTx(ctx, tx, balance); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}

	master.AccountBalance = balance
	return toDomainAccount(master), nil
}

func (s *accountService) InitializeAccount(ctx context.Context, subjectID string, idempotencyKey *uuid.UUID) (domain.Account, *domain.BankAccountError) {
	var master *entity.AccountMaster
	err := s.withTx(ctx, func(tx *gorm.DB) error {
		var err error
		master, err = s.getMasterForUpdate(ctx, tx, subjectID)
		if err != nil {
			return err
		}

		if duplicate, err := s.checkIdempotency(ctx, tx, idempotencyKey); err != nil {
			return err
		} else if duplicate {
			return nil
		}

		if master.AccountBalance.LoanPrincipal > 0 {
			return domain.NewBankAccountError(domain.ErrorTypeAlreadyExists, "account already initialized with loan")
		}

		loanAmount := int64(1000000)
		master.AccountBalance.Balance += loanAmount
		master.AccountBalance.LoanPrincipal += loanAmount

		if _, err := s.accountBalanceRepository.UpdateAccountBalanceTx(ctx, tx, master.AccountBalance); err != nil {
			return err
		}

		t := &entity.Transaction{
			UserID:         master.UserID,
			Type:           "LOAN",
			Amount:         loanAmount,
			BalanceAfter:   master.AccountBalance.Balance,
			Description:    "初期ローン",
			IsPrinted:      false,
			IdempotencyKey: idempotencyKey,
		}
		if _, err := s.transactionRepository.CreateTransactionTx(ctx, tx, t); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}

	return toDomainAccount(master), nil
}

func (s *accountService) GetAccountHistory(ctx context.Context, subjectID string) ([]domain.Transaction, *domain.BankAccountError) {
	master, err := s.accountRepository.GetMasterByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return nil, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}
	if master == nil {
		return nil, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account master is nil")
	}

	list, err := s.transactionRepository.GetTransactionsByUserID(ctx, master.UserID)
	if err != nil {
		return nil, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return toDomainTransactions(list), nil
}

func (s *accountService) MarkAsPrinted(ctx context.Context, subjectID string, ids []int64) *domain.BankAccountError {
	_, err := s.accountRepository.GetMasterByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return s.withTx(ctx, func(tx *gorm.DB) error {
		return s.transactionRepository.MarkAsPrintedTx(ctx, tx, ids)
	})
}

func (s *accountService) SettleAccount(ctx context.Context, subjectID string) (domain.Account, *domain.BankAccountError) {
	var master *entity.AccountMaster
	err := s.withTx(ctx, func(tx *gorm.DB) error {
		var err error
		master, err = s.getMasterForUpdate(ctx, tx, subjectID)
		if err != nil {
			return err
		}

		netAsset := master.AccountBalance.Balance - master.AccountBalance.LoanPrincipal
		score := master.CreditScore
		if netAsset >= 0 {
			score += 1
		} else {
			score -= 2
		}

		if score > 10 {
			score = 10
		}
		if score < 1 {
			score = 1
		}

		master.CreditScore = score
		master.IsFrozen = (score <= 1)

		if _, err := s.accountRepository.UpdateMasterTx(ctx, tx, master); err != nil {
			return err
		}

		t := &entity.Transaction{
			UserID:       master.UserID,
			Type:         "SETTLE",
			Amount:       0,
			BalanceAfter: master.AccountBalance.Balance,
			Description:  "最終精算",
			IsPrinted:    false,
		}
		if _, err := s.transactionRepository.CreateTransactionTx(ctx, tx, t); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}

	return toDomainAccount(master), nil
}

func (s *accountService) ExecuteTransaction(ctx context.Context, subjectID string, amount int64, txType string, desc string, idempotencyKey *uuid.UUID) (domain.Account, *domain.BankAccountError) {
	var master *entity.AccountMaster
	err := s.withTx(ctx, func(tx *gorm.DB) error {
		var err error
		master, err = s.getMasterForUpdate(ctx, tx, subjectID)
		if err != nil {
			return err
		}
		if master.IsFrozen {
			return domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account is frozen")
		}

		if duplicate, err := s.checkIdempotency(ctx, tx, idempotencyKey); err != nil {
			return err
		} else if duplicate {
			return nil
		}

		actualAmount := amount
		if txType == "REPAYMENT" {
			repayAmount := -amount
			if repayAmount > master.AccountBalance.LoanPrincipal {
				repayAmount = master.AccountBalance.LoanPrincipal
			}
			master.AccountBalance.Balance -= repayAmount
			master.AccountBalance.LoanPrincipal -= repayAmount
			actualAmount = -repayAmount
		} else {
			master.AccountBalance.Balance += amount
		}

		if _, err := s.accountBalanceRepository.UpdateAccountBalanceTx(ctx, tx, master.AccountBalance); err != nil {
			return err
		}

		t := &entity.Transaction{
			UserID:         master.UserID,
			Type:           txType,
			Amount:         actualAmount,
			BalanceAfter:   master.AccountBalance.Balance,
			Description:    desc,
			IsPrinted:      false,
			IdempotencyKey: idempotencyKey,
		}
		if _, err := s.transactionRepository.CreateTransactionTx(ctx, tx, t); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}

	return toDomainAccount(master), nil
}

func (s *accountService) ApplyInterestBatch(ctx context.Context) *domain.BankAccountError {
	limit := 100
	offset := 0
	ns := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	for {
		masters, err := s.accountRepository.GetMastersPage(ctx, limit, offset)
		if err != nil {
			return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
		}
		if len(masters) == 0 {
			break
		}

		berr := s.withTx(ctx, func(tx *gorm.DB) error {
			for i := range masters {
				m := &masters[i]
				if m.AccountBalance == nil || m.IsFrozen {
					continue
				}
				if m.AccountBalance.LoanPrincipal > 0 {
					data := m.UserID.String() + "-interest-turn-" + strconv.Itoa(m.CurrentTurn)
					idempKey := uuid.NewSHA1(ns, []byte(data))

					duplicate, err := s.checkIdempotency(ctx, tx, &idempKey)
					if err != nil {
						return err
					}
					if duplicate {
						continue
					}

					interest := int64(float64(m.AccountBalance.LoanPrincipal) * 0.10)
					if interest > 0 {
						m.AccountBalance.LoanPrincipal += interest
						if _, err := s.accountBalanceRepository.UpdateAccountBalanceTx(ctx, tx, m.AccountBalance); err != nil {
							return err
						}

						t := &entity.Transaction{
							UserID:         m.UserID,
							Type:           "INTEREST",
							Amount:         0,
							BalanceAfter:   m.AccountBalance.Balance,
							Description:    "利息加算",
							IsPrinted:      false,
							IdempotencyKey: &idempKey,
						}
						if _, err := s.transactionRepository.CreateTransactionTx(ctx, tx, t); err != nil {
							return err
						}
					}
				}
			}
			return nil
		})
		if berr != nil {
			return berr
		}

		if len(masters) < limit {
			break
		}
		offset += limit
	}

	return nil
}

func (s *accountService) ReconcileAccountsBatch(ctx context.Context) ([]uuid.UUID, *domain.BankAccountError) {
	var inconsistentUserIDs []uuid.UUID
	limit := 100
	offset := 0

	for {
		masters, err := s.accountRepository.GetMastersPage(ctx, limit, offset)
		if err != nil {
			return nil, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
		}
		if len(masters) == 0 {
			break
		}

		berr := s.withTx(ctx, func(tx *gorm.DB) error {
			for i := range masters {
				m := &masters[i]
				if m.AccountBalance == nil {
					continue
				}

				sum, err := s.transactionRepository.GetTransactionSumByUserID(ctx, m.UserID)
				if err != nil {
					return err
				}

				if sum != m.AccountBalance.Balance {
					inconsistentUserIDs = append(inconsistentUserIDs, m.UserID)
					m.IsFrozen = true
					if _, err := s.accountRepository.UpdateMasterTx(ctx, tx, m); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if berr != nil {
			return nil, berr
		}

		if len(masters) < limit {
			break
		}
		offset += limit
	}

	return inconsistentUserIDs, nil
}
