package echobind

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/proemergotech/bind/internal"
)

func JSONContentTypeMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(eCtx echo.Context) error {
			req := eCtx.Request()

			contentType := req.Header.Get(echo.HeaderContentType)
			if !strings.HasPrefix(contentType, echo.MIMEApplicationJSON) {
				return internal.JSONContentTypeError{ContentType: contentType}.E()
			}

			return next(eCtx)
		}
	}
}
