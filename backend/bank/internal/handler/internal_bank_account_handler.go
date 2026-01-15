package handler

import (
	"github/t-takamichi/fintech-game/backend/bank/internal/service"
	"net/http"

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

func (h *InternalBankAccountHandler) Create(c echo.Context) error {

	var req RequestCreateAccount
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	context := c.Request().Context()
	account, err := h.svc.CreateAccount(context, req.SubjectID, req.InitialScore)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})

	}
	return c.JSON(http.StatusCreated, account)
}
