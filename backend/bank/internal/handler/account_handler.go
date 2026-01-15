package handler

import (
	"net/http"

	"github/t-takamichi/fintech-game/backend/bank/internal/service"

	"github.com/labstack/echo/v4"
)

type AccountHandler struct {
	svc service.AccountService
}

func NewAccountHandler(svc service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) GetAccountStatusHandler(c echo.Context) error {
	subjectID := c.Param("id")
	status, err := h.svc.GetAccountStatus(subjectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, status)
}
