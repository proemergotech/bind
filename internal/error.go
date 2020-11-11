package internal

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"gitlab.com/proemergotech/errors"
)

const (
	ErrCode     = "code"
	ErrHTTPCode = "http_code"

	ErrOnlyJSONContentTypeAllowed = "ERR_ONLY_JSON_CONTENT_TYPE_ALLOWED"
)

type JSONContentTypeError struct {
	ContentType string
}

func (e JSONContentTypeError) E() error {
	msg := fmt.Sprintf("only '%s' content type allowed, got: '%s'", echo.MIMEApplicationJSON, e.ContentType)

	return errors.WithFields(
		errors.New(msg),
		ErrHTTPCode, 400,
		ErrCode, ErrOnlyJSONContentTypeAllowed,
	)
}
