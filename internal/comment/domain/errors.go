package domain

import "errors"

var (
	ErrInvalidContent = errors.New("comment: invalid content")
	ErrInvalidPost    = errors.New("comment: invalid post id")
	ErrInvalidAuthor  = errors.New("comment: invalid author id")

	ErrNotFound  = errors.New("comment: not found")
	ErrForbidden = errors.New("comment: caller is not permitted to perform this action")
)
