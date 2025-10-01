package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// Logger returns a middleware that logs HTTP requests using slog
func Logger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Log request
			req := c.Request()
			res := c.Response()

			logger.Info("http request",
				"method", req.Method,
				"uri", req.RequestURI,
				"status", res.Status,
				"latency_ms", time.Since(start).Milliseconds(),
				"remote_ip", c.RealIP(),
				"user_agent", req.UserAgent(),
			)

			return err
		}
	}
}
