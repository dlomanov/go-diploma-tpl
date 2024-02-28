package entity

import (
	"fmt"
	"time"
)

var (
	_ error = (*AppError)(nil)
	_ error = (*RetryError)(nil)
)

type (
	AppError struct {
		msg string
	}
	RetryError struct {
		AppError
		RetryAfter time.Duration
	}
)

func (e AppError) Error() string {
	return e.msg
}

func (e RetryError) Error() string {
	return fmt.Sprintf("%s, retry after %s", e.msg, e.RetryAfter.String())
}
