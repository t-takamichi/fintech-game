package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type InternalBankAccountHandler struct {
}

func NewInternalBankAccountHandler() *InternalBankAccountHandler {
	return &InternalBankAccountHandler{}
}

func (h *InternalBankAccountHandler) Create(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}
