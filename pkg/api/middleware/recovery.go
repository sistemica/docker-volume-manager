package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

// Recovery returns a middleware that recovers from panics
func Recovery(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					logger.Error("panic recovered",
						"error", err,
						"stack", string(debug.Stack()),
					)

					c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Internal server error",
					})
				}
			}()

			return next(c)
		}
	}
}
