package content

import "errors"

var (
	ErrNotFound        = errors.New("content: document not found")
	ErrUnauthorized    = errors.New("content: unauthorized")
	ErrForbidden       = errors.New("content: forbidden")
	ErrRateLimited     = errors.New("content: rate limited")
	ErrUnavailable     = errors.New("content: unavailable")
	ErrInvalidResponse = errors.New("content: invalid response")
	ErrConflict        = errors.New("content: conflict")
)
