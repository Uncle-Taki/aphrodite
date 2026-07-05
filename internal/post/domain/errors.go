package domain

import "errors"

var (
	ErrInvalidTitle   = errors.New("post: invalid title")
	ErrInvalidContent = errors.New("post: invalid content")
	ErrInvalidAuthor  = errors.New("post: invalid author id")

	ErrNotFound  = errors.New("post: not found")
	ErrForbidden = errors.New("post: caller is not permitted to perform this action")
)
