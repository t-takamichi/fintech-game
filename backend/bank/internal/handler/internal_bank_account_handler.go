package handler

import (
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	"github/t-takamichi/fintech-game/backend/bank/internal/service"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RequestCreateAccount struct {
	SubjectID    string `json:"subject_id"`
	InitialScore int    `json:"initial_score"`
}

type InternalBankAccountHandler struct {
	svc service.AccountService
}

func NewInternalBankAccountHandler(svc service.AccountService) *InternalBankAccountHandler {
	return &InternalBankAccountHandler{svc: svc}
}

func parseIdempotencyKey(keyStr string) (*uuid.UUID, error) {
	if keyStr == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(keyStr)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (h *InternalBankAccountHandler) Create(c echo.Context) error {
	var req RequestCreateAccount
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.SubjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject_id is required"})
	}
	initialScore := req.InitialScore
	if initialScore == 0 {
		initialScore = 3
	}
	if initialScore < 1 || initialScore > 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "initial_score must be between 1 and 10"})
	}

	ctx := c.Request().Context()
	account, bankErr := h.svc.CreateAccount(ctx, req.SubjectID, initialScore)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeAlreadyExists:
			return c.JSON(http.StatusConflict, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.JSON(http.StatusCreated, account)
}

type RequestInitializeAccount struct {
	SubjectID      string `json:"subject_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *InternalBankAccountHandler) Initialize(c echo.Context) error {
	var req RequestInitializeAccount
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.SubjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject_id is required"})
	}

	idempotencyKey, err := parseIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid idempotency_key format"})
	}

	ctx := c.Request().Context()
	account, bankErr := h.svc.InitializeAccount(ctx, req.SubjectID, idempotencyKey)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeNotFound:
			return c.JSON(http.StatusNotFound, bankErr)
		case domain.ErrorTypeAlreadyExists:
			return c.JSON(http.StatusConflict, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.JSON(http.StatusOK, account)
}

type RequestSettleAccount struct {
	SubjectID string `json:"subject_id"`
}

func (h *InternalBankAccountHandler) Settle(c echo.Context) error {
	var req RequestSettleAccount
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.SubjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject_id is required"})
	}

	ctx := c.Request().Context()
	account, bankErr := h.svc.SettleAccount(ctx, req.SubjectID)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeNotFound:
			return c.JSON(http.StatusNotFound, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.JSON(http.StatusOK, account)
}

type RequestExecuteTransaction struct {
	SubjectID      string `json:"subject_id"`
	Amount         int64  `json:"amount"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *InternalBankAccountHandler) ExecuteTransaction(c echo.Context) error {
	var req RequestExecuteTransaction
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.SubjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject_id is required"})
	}
	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "type is required"})
	}

	idempotencyKey, err := parseIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid idempotency_key format"})
	}

	ctx := c.Request().Context()
	account, bankErr := h.svc.ExecuteTransaction(ctx, req.SubjectID, req.Amount, req.Type, req.Description, idempotencyKey)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeNotFound:
			return c.JSON(http.StatusNotFound, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.JSON(http.StatusOK, account)
}

func (h *InternalBankAccountHandler) ApplyInterest(c echo.Context) error {
	ctx := c.Request().Context()
	bankErr := h.svc.ApplyInterestBatch(ctx)
	if bankErr != nil {
		return c.JSON(http.StatusInternalServerError, bankErr)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "interest batch applied successfully"})
}

func (h *InternalBankAccountHandler) Reconcile(c echo.Context) error {
	ctx := c.Request().Context()
	inconsistentUserIDs, bankErr := h.svc.ReconcileAccountsBatch(ctx)
	if bankErr != nil {
		return c.JSON(http.StatusInternalServerError, bankErr)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":               "reconciliation batch executed successfully",
		"inconsistent_user_ids": inconsistentUserIDs,
	})
}
