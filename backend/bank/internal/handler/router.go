package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, accountHandler *AccountHandler, internalBankAccountHandler *InternalBankAccountHandler) {
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	bank := e.Group("/api/bank")
	bank.GET("/account/:id/status", accountHandler.GetAccountStatusHandler)
	bank.GET("/account/:id/history", accountHandler.GetAccountHistoryHandler)
	bank.PATCH("/account/:id/history/print", accountHandler.MarkAsPrintedHandler)

	internal := e.Group("/internal/bank-accounts")
	internal.POST("/create", internalBankAccountHandler.Create)
	internal.POST("/initialize", internalBankAccountHandler.Initialize)
	internal.POST("/settle", internalBankAccountHandler.Settle)
	internal.POST("/transaction/execute", internalBankAccountHandler.ExecuteTransaction)
	internal.POST("/batch/interest", internalBankAccountHandler.ApplyInterest)
	internal.POST("/batch/reconcile", internalBankAccountHandler.Reconcile)
}
