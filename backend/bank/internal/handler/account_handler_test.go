package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github/t-takamichi/fintech-game/backend/bank/internal/db"
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	repository "github/t-takamichi/fintech-game/backend/bank/internal/repository"
	"github/t-takamichi/fintech-game/backend/bank/internal/service"

	"github.com/labstack/echo/v4"
)

func setupTestHandler(t *testing.T) (service.AccountService, *AccountHandler, *InternalBankAccountHandler) {
	t.Helper()
	gdb, err := db.InitDBFromEnv()
	if err != nil {
		t.Fatalf("failed to connect DB: %v", err)
	}

	if err := gdb.Exec("TRUNCATE TABLE transactions, accounts_balance, accounts_master CASCADE").Error; err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	accountRepo := repository.NewAccountRepository(gdb)
	balanceRepo := repository.NewAccountBalanceRepository(gdb)
	txRepo := repository.NewTransactionRepository(gdb)

	svc := service.NewAccountService(accountRepo, balanceRepo, txRepo, gdb)
	h := NewAccountHandler(svc)
	ih := NewInternalBankAccountHandler(svc)

	return svc, h, ih
}

func TestAccountHandler_GetAccountStatus(t *testing.T) {
	svc, h, _ := setupTestHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/api/bank/account/non-existent/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/bank/account/:id/status")
	c.SetParamNames("id")
	c.SetParamValues("non-existent")

	err := h.GetAccountStatusHandler(c)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	ctx := context.Background()
	_, berr := svc.CreateAccount(ctx, "test-user", 3)
	if berr != nil {
		t.Fatalf("failed to create account: %v", berr)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/bank/account/test-user/status", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/api/bank/account/:id/status")
	c.SetParamNames("id")
	c.SetParamValues("test-user")

	err = h.GetAccountStatusHandler(c)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var status domain.AccountStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if status.CreditScore != 3 {
		t.Errorf("expected score 3, got %d", status.CreditScore)
	}
}

func TestAccountHandler_MarkAsPrinted(t *testing.T) {
	_, h, _ := setupTestHandler(t)
	e := echo.New()

	reqBody := `{"ids": [1, 2]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/bank/account/test-user/history/print", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.SetPath("/api/bank/account/:id/history/print")
	c.SetParamNames("id")
	c.SetParamValues("test-user")

	_, berr := h.svc.CreateAccount(context.Background(), "test-user", 3)
	if berr != nil {
		t.Fatalf("failed to create test-user: %v", berr)
	}

	err := h.MarkAsPrintedHandler(c)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
}

func TestAccountHandler_GetAccountHistory(t *testing.T) {
	svc, h, _ := setupTestHandler(t)
	e := echo.New()

	ctx := context.Background()
	_, berr := svc.CreateAccount(ctx, "test-user", 3)
	if berr != nil {
		t.Fatalf("failed to create account: %v", berr)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/bank/account/test-user/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/bank/account/:id/history")
	c.SetParamNames("id")
	c.SetParamValues("test-user")

	err := h.GetAccountHistoryHandler(c)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var list []domain.Transaction
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 transactions initially, got %d", len(list))
	}
}
