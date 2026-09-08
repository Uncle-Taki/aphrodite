package strapi

import (
	"fmt"

	"aphrodite/internal/content"
)

var (
	ErrNotFound        = content.ErrNotFound
	ErrUnauthorized    = content.ErrUnauthorized
	ErrForbidden       = content.ErrForbidden
	ErrRateLimited     = content.ErrRateLimited
	ErrUnavailable     = content.ErrUnavailable
	ErrInvalidResponse = content.ErrInvalidResponse
	ErrConflict        = content.ErrConflict
)

type HTTPError struct {
	Status  int
	Name    string
	Message string
}

func (e *HTTPError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("strapi: HTTP %d", e.Status)
	}
	return fmt.Sprintf("strapi: HTTP %d: %s", e.Status, e.Message)
}

func (e *HTTPError) Unwrap() error {
	switch e.Status {
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 409:
		return ErrConflict
	case 429:
		return ErrRateLimited
	case 500, 502, 503, 504:
		return ErrUnavailable
	default:
		return nil
	}
}
