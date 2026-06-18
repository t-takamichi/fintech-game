package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	db "github/t-takamichi/fintech-game/backend/bank/internal/db"
	"github/t-takamichi/fintech-game/backend/bank/internal/handler"
	repository "github/t-takamichi/fintech-game/backend/bank/internal/repository"
	svc "github/t-takamichi/fintech-game/backend/bank/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	echoServer := echo.New()
	echoServer.HTTPErrorHandler = handler.CustomHTTPErrorHandler

	gdb, err := db.InitDBFromEnv()
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		return
	}

	accountRepository := repository.NewAccountRepository(gdb)
	accountBalanceRepository := repository.NewAccountBalanceRepository(gdb)
	transactionRepository := repository.NewTransactionRepository(gdb)

	accountService := svc.NewAccountService(accountRepository, accountBalanceRepository, transactionRepository, gdb)
	accountHandler := handler.NewAccountHandler(accountService)
	internalBankAccountHandler := handler.NewInternalBankAccountHandler(accountService)

	echoServer.Use(middleware.Recover())
	echoServer.Use(middleware.RequestLogger())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler.RegisterRoutes(echoServer, accountHandler, internalBankAccountHandler)

	if err := echoServer.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}
}
