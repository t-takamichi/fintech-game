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

// DebtThreshold は口座が債務（負債）と見なされる閾値です。
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
	uuidGenerator            func() uuid.UUID // 外部注入可能なUUIDジェネレータ
}

func NewAccountService(r repository.AccountRepository, b repository.AccountBalanceRepository, t repository.TransactionRepository, db *gorm.DB) AccountService {
	return &accountService{
		accountRepository:        r,
		accountBalanceRepository: b,
		transactionRepository:    t,
		db:                       db,
		uuidGenerator:            uuid.New, // デフォルトで uuid.New を使用
	}
}

func (s *accountService) GetAccountStatus(ctx context.Context, subjectID string) (domain.AccountStatus, *domain.BankAccountError) {
	m, err := s.accountRepository.GetMasterByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}
	if m == nil {
		return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account master is nil")
	}

	if m.AccountBalance == nil {
		return domain.AccountStatus{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account balance is nil")
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

func (s *accountService) CreateAccount(ctx context.Context, subjectID string, initialScore int) (domain.Account, *domain.BankAccountError) {
	// 口座作成パラメータのバリデーション
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
	if verr != nil && !errors.Is(verr, gorm.ErrRecordNotFound) {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, verr.Error())
	}

	id := s.uuidGenerator()
	master := &entity.AccountMaster{UserID: id, SubjectID: subjectID, CreditScore: initialScore, IsFrozen: false, CurrentTurn: 0}
	balance := &entity.AccountBalance{UserID: id, Balance: 0, LoanPrincipal: 0}

	var created *entity.AccountMaster
	// トランザクションの範囲は、複数リポジトリの整合性を保つためサービス層で制御する設計方針とする。
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.accountRepository.CreateMasterTx(ctx, tx, master); err != nil {
			return err
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
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return toDomainAccount(created), nil
}

func (s *accountService) InitializeAccount(ctx context.Context, subjectID string, idempotencyKey *uuid.UUID) (domain.Account, *domain.BankAccountError) {
	var updatedMaster *entity.AccountMaster
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE ロックをかけてマスター取得（並行処理からのデータ競合を防止）
		master, err := s.accountRepository.GetMasterForUpdateTx(ctx, tx, subjectID)
		if err != nil {
			return err
		}

		// べき等性の検証
		if idempotencyKey != nil {
			exist, err := s.transactionRepository.GetTransactionByIdempotencyKeyTx(ctx, tx, *idempotencyKey)
			if err == nil && exist != nil {
				// すでに同一キーで登録済みの場合は、何もせず現在の状態をロードして終了
				updatedMaster = master
				return nil
			}
		}

		// すでに初期ローンが実行されているか確認
		if master.AccountBalance.LoanPrincipal > 0 {
			return domain.NewBankAccountError(domain.ErrorTypeAlreadyExists, "account already initialized with loan")
		}

		loanAmount := int64(1000000)
		master.AccountBalance.Balance += loanAmount
		master.AccountBalance.LoanPrincipal += loanAmount

		if _, err := s.accountBalanceRepository.UpdateAccountBalanceTx(ctx, tx, master.AccountBalance); err != nil {
			return err
		}

		// transaction 履歴を保存
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

		// 最新データをロード
		if err := tx.Preload("AccountBalance").First(&updatedMaster, "user_id = ?", master.UserID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		var berr *domain.BankAccountError
		if errors.As(err, &berr) {
			return domain.Account{}, berr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return toDomainAccount(updatedMaster), nil
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
	master, err := s.accountRepository.GetMasterByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}
	if master == nil {
		return domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account master is nil")
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		return s.transactionRepository.MarkAsPrintedTx(ctx, tx, ids)
	})
	if err != nil {
		return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return nil
}

func (s *accountService) SettleAccount(ctx context.Context, subjectID string) (domain.Account, *domain.BankAccountError) {
	master, err := s.accountRepository.GetMasterByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}
	if master == nil {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account master is nil")
	}
	if master.AccountBalance == nil {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account balance is nil")
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

	isFrozen := score <= 1

	var updatedMaster *entity.AccountMaster
	err = s.db.Transaction(func(tx *gorm.DB) error {
		master.CreditScore = score
		master.IsFrozen = isFrozen

		if _, err := s.accountRepository.UpdateMasterTx(ctx, tx, master); err != nil {
			return err
		}

		// SETTLE 取引を記録
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

		if err := tx.Preload("AccountBalance").First(&updatedMaster, "user_id = ?", master.UserID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return toDomainAccount(updatedMaster), nil
}

func (s *accountService) ExecuteTransaction(ctx context.Context, subjectID string, amount int64, txType string, desc string, idempotencyKey *uuid.UUID) (domain.Account, *domain.BankAccountError) {
	var updatedMaster *entity.AccountMaster
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE ロックをかけてマスター取得（並行アクセス時の競合を防ぐ）
		master, err := s.accountRepository.GetMasterForUpdateTx(ctx, tx, subjectID)
		if err != nil {
			return err
		}

		if master.IsFrozen {
			return domain.NewBankAccountError(domain.ErrorTypeInconsistent, "account is frozen")
		}

		// べき等性の検証
		if idempotencyKey != nil {
			exist, err := s.transactionRepository.GetTransactionByIdempotencyKeyTx(ctx, tx, *idempotencyKey)
			if err == nil && exist != nil {
				// すでに同一キーで登録済みの場合は、何もせず現在の状態をロードして終了
				updatedMaster = master
				return nil
			}
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

		if err := tx.Preload("AccountBalance").First(&updatedMaster, "user_id = ?", master.UserID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		var berr *domain.BankAccountError
		if errors.As(err, &berr) {
			return domain.Account{}, berr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeNotFound, "account not found")
		}
		return domain.Account{}, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
	}

	return toDomainAccount(updatedMaster), nil
}

func (s *accountService) ApplyInterestBatch(ctx context.Context) *domain.BankAccountError {
	limit := 100
	offset := 0
	ns := uuid.MustParse("00000000-0000-0000-0000-000000000000") // 固定ネームスペース

	for {
		masters, err := s.accountRepository.GetMastersPage(ctx, limit, offset)
		if err != nil {
			return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
		}
		if len(masters) == 0 {
			break
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			for _, m := range masters {
				if m.AccountBalance == nil || m.IsFrozen {
					continue
				}
				if m.AccountBalance.LoanPrincipal > 0 {
					// 決定論的べき等キーの生成
					data := m.UserID.String() + "-interest-turn-" + strconv.Itoa(m.CurrentTurn)
					idempKey := uuid.NewSHA1(ns, []byte(data))

					// すでに同一キーの利息加算取引が存在するかチェック
					exist, err := s.transactionRepository.GetTransactionByIdempotencyKeyTx(ctx, tx, idempKey)
					if err == nil && exist != nil {
						// すでに適用済みの場合はスキップ
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
		if err != nil {
			return domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
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

		err = s.db.Transaction(func(tx *gorm.DB) error {
			for _, m := range masters {
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
					if _, err := s.accountRepository.UpdateMasterTx(ctx, tx, &m); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, domain.NewBankAccountError(domain.ErrorTypeInconsistent, err.Error())
		}

		if len(masters) < limit {
			break
		}
		offset += limit
	}

	return inconsistentUserIDs, nil
}
