package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCustomHTTPErrorHandler(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	errGeneral := errors.New("something went wrong")
	CustomHTTPErrorHandler(errGeneral, c)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Internal Server Error") {
		t.Errorf("expected body to contain 'Internal Server Error', got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	errHTTP := echo.NewHTTPError(http.StatusNotFound, "custom message not found")
	CustomHTTPErrorHandler(errHTTP, c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "custom message not found") {
		t.Errorf("expected body to contain 'custom message not found', got %s", rec.Body.String())
	}
}
