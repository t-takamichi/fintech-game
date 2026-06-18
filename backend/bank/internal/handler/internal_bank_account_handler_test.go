package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github/t-takamichi/fintech-game/backend/bank/internal/domain"

	"github.com/labstack/echo/v4"
)

func TestInternalBankAccountHandler_Create(t *testing.T) {
	_, _, ih := setupTestHandler(t)
	e := echo.New()

	reqBody := `{"subject_id": "test-user-1", "initial_score": 5}`
	req := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/create", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := ih.Create(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/create", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = ih.Create(c)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}
}

func TestInternalBankAccountHandler_Initialize(t *testing.T) {
	_, _, ih := setupTestHandler(t)
	e := echo.New()

	reqBodyCreate := `{"subject_id": "test-user-1"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/create", strings.NewReader(reqBodyCreate))
	reqCreate.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recCreate := httptest.NewRecorder()
	cCreate := e.NewContext(reqCreate, recCreate)
	if err := ih.Create(cCreate); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	reqBodyInit := `{"subject_id": "test-user-1"}`
	reqInit := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/initialize", strings.NewReader(reqBodyInit))
	reqInit.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recInit := httptest.NewRecorder()
	cInit := e.NewContext(reqInit, recInit)

	err := ih.Initialize(cInit)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if recInit.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recInit.Code)
	}
}

func TestInternalBankAccountHandler_ExecuteTransaction(t *testing.T) {
	_, _, ih := setupTestHandler(t)
	e := echo.New()

	reqBodyCreate := `{"subject_id": "test-user-1"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/create", strings.NewReader(reqBodyCreate))
	reqCreate.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recCreate := httptest.NewRecorder()
	cCreate := e.NewContext(reqCreate, recCreate)
	if err := ih.Create(cCreate); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	reqBodyInit := `{"subject_id": "test-user-1"}`
	reqInit := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/initialize", strings.NewReader(reqBodyInit))
	reqInit.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recInit := httptest.NewRecorder()
	cInit := e.NewContext(reqInit, recInit)
	if err := ih.Initialize(cInit); err != nil {
		t.Fatalf("failed to initialize account: %v", err)
	}

	reqBodyTx := `{"subject_id": "test-user-1", "amount": -200000, "type": "BUY", "description": "ロケット株"}`
	reqTx := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/transaction/execute", strings.NewReader(reqBodyTx))
	reqTx.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recTx := httptest.NewRecorder()
	cTx := e.NewContext(reqTx, recTx)

	err := ih.ExecuteTransaction(cTx)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if recTx.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recTx.Code)
	}

	var acc domain.Account
	if err := json.Unmarshal(recTx.Body.Bytes(), &acc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if acc.Balance != 800000 {
		t.Errorf("expected balance 800,000, got %d", acc.Balance)
	}
}

func TestInternalBankAccountHandler_Batches(t *testing.T) {
	_, _, ih := setupTestHandler(t)
	e := echo.New()

	reqInt := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/batch/interest", nil)
	recInt := httptest.NewRecorder()
	cInt := e.NewContext(reqInt, recInt)

	err := ih.ApplyInterest(cInt)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if recInt.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recInt.Code)
	}

	reqRec := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/batch/reconcile", nil)
	recRec := httptest.NewRecorder()
	cRec := e.NewContext(reqRec, recRec)

	err = ih.Reconcile(cRec)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if recRec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recRec.Code)
	}
}

func TestInternalBankAccountHandler_Settle(t *testing.T) {
	_, _, ih := setupTestHandler(t)
	e := echo.New()

	// 口座作成
	reqBodyCreate := `{"subject_id": "test-user-1"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/create", strings.NewReader(reqBodyCreate))
	reqCreate.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recCreate := httptest.NewRecorder()
	cCreate := e.NewContext(reqCreate, recCreate)
	if err := ih.Create(cCreate); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// 精算実行
	reqBodySettle := `{"subject_id": "test-user-1"}`
	reqSettle := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/settle", strings.NewReader(reqBodySettle))
	reqSettle.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recSettle := httptest.NewRecorder()
	cSettle := e.NewContext(reqSettle, recSettle)

	err := ih.Settle(cSettle)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if recSettle.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recSettle.Code)
	}

	var acc domain.Account
	if err := json.Unmarshal(recSettle.Body.Bytes(), &acc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// 初期信用スコア3 => 精算成功で4になるはず
	if acc.CreditScore != 4 {
		t.Errorf("expected credit score 4, got %d", acc.CreditScore)
	}
}
