package handler

import (
	"github/t-takamichi/fintech-game/backend/bank/internal/domain"
	"github/t-takamichi/fintech-game/backend/bank/internal/service"
	"net/http"

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
	if subjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id parameter is required"})
	}

	ctx := c.Request().Context()
	status, bankErr := h.svc.GetAccountStatus(ctx, subjectID)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeNotFound:
			return c.JSON(http.StatusNotFound, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.JSON(http.StatusOK, status)
}

func (h *AccountHandler) GetAccountHistoryHandler(c echo.Context) error {
	subjectID := c.Param("id")
	if subjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id parameter is required"})
	}

	ctx := c.Request().Context()
	list, bankErr := h.svc.GetAccountHistory(ctx, subjectID)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeNotFound:
			return c.JSON(http.StatusNotFound, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.JSON(http.StatusOK, list)
}

type RequestMarkAsPrinted struct {
	IDs []int64 `json:"ids"`
}

func (h *AccountHandler) MarkAsPrintedHandler(c echo.Context) error {
	subjectID := c.Param("id")
	if subjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id parameter is required"})
	}

	var req RequestMarkAsPrinted
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ids parameter is required"})
	}

	ctx := c.Request().Context()
	bankErr := h.svc.MarkAsPrinted(ctx, subjectID, req.IDs)
	if bankErr != nil {
		switch bankErr.Type {
		case domain.ErrorTypeNotFound:
			return c.JSON(http.StatusNotFound, bankErr)
		default:
			return c.JSON(http.StatusInternalServerError, bankErr)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
