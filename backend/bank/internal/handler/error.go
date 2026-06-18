package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	var msg interface{} = "Internal Server Error"

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		msg = he.Message
	}

	c.Logger().Error(err)
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, map[string]interface{}{
				"error": msg,
			})
		}
		if err != nil {
			c.Logger().Error(err)
		}
	}
}
