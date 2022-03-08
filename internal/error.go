package internal

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/proemergotech/errors/v2"
)

const (
	ErrCode     = "code"
	ErrHTTPCode = "http_code"

	ErrOnlyJSONContentTypeAllowed = "ERR_ONLY_JSON_CONTENT_TYPE_ALLOWED"
	ErrContentTypeWithoutBody     = "ERR_CONTENT_TYPE_WITHOUT_BODY"
	ErrCannotReadBody             = "ERR_CANNOT_READ_BODY"
)

type ContentTypeWithoutBodyError struct{}

func (e ContentTypeWithoutBodyError) E() error {
	msg := "empty body: content type header only allowed with body"

	return errors.WithFields(
		errors.New(msg),
		ErrHTTPCode, 400,
		ErrCode, ErrContentTypeWithoutBody,
	)
}

type CannotReadBodyError struct{}

func (e CannotReadBodyError) E() error {
	return errors.WithFields(
		errors.New("cannot read body"),
		ErrHTTPCode, 500,
		ErrCode, ErrCannotReadBody,
	)
}

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
