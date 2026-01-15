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

	echo := echo.New()

	// FIXME: DB接続まわり初期化処理を分離したい
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("POSTGRES_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("POSTGRES_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("POSTGRES_USER")
		if user == "" {
			user = "bank_user"
		}
		pass := os.Getenv("POSTGRES_PASSWORD")
		if pass == "" {
			pass = "bank_pass"
		}
		dbname := os.Getenv("POSTGRES_DB")
		if dbname == "" {
			dbname = "bank_db"
		}
		dsn = "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	}

	gdb, err := db.NewGormDB(dsn)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		return
	}

	// create repository layer backed by GORM
	accountRepository := repository.NewAccountRepository(gdb)
	accountBalanceRepository := repository.NewAccountBalanceRepository(gdb)

	accountService := svc.NewAccountService(accountRepository, accountBalanceRepository, gdb)
	accountHandler := handler.NewAccountHandler(accountService)

	internalBankAccountHandler := handler.NewInternalBankAccountHandler(accountService)

	echo.Use(middleware.Recover())
	echo.Use(middleware.RequestLogger())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// FIXME: ルーティング設定を分離したい

	echo.GET("/", hello)

	bank := echo.Group("/api/bank")
	bank.GET("/account/:id/status", accountHandler.GetAccountStatusHandler)

	internal := echo.Group("/internal/bank-accounts")
	internal.POST("/Create", internalBankAccountHandler.Create)

	// TODO: グローバルでエラーハンドリングできるようにしたい

	// Start server
	if err := echo.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}
}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
