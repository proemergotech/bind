package echobind

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/proemergotech/bind/internal"
)

type settings struct {
	readBody bool
}

type Option func(*settings)

func ReadBody() Option {
	return func(opts *settings) {
		opts.readBody = true
	}
}

func JSONContentTypeMiddleware(options ...Option) echo.MiddlewareFunc {
	s := &settings{}

	for _, opt := range options {
		opt(s)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(eCtx echo.Context) error {
			req := eCtx.Request()
			contentType := req.Header.Get(echo.HeaderContentType)

			if req.Body == nil || req.Body == http.NoBody || req.ContentLength == 0 || req.Method == http.MethodGet {
				if contentType != "" {
					return internal.ContentTypeWithoutBodyError{}.E()
				}
				return next(eCtx)
			}

			if s.readBody && req.ContentLength == -1 {
				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					return internal.CannotReadBodyError{}.E()
				}
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				if string(body) == "" {
					if contentType != "" {
						return internal.ContentTypeWithoutBodyError{}.E()
					}
					return next(eCtx)
				}
			}

			if !strings.HasPrefix(contentType, echo.MIMEApplicationJSON) {
				return internal.JSONContentTypeError{ContentType: contentType}.E()
			}

			return next(eCtx)
		}
	}
}
