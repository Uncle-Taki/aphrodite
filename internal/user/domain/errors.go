package domain

import "errors"

var (
	ErrInvalidUsername     = errors.New("user: invalid username")
	ErrInvalidEmail        = errors.New("user: invalid email")
	ErrInvalidPasswordHash = errors.New("user: invalid password hash")
	ErrInvalidRole         = errors.New("user: invalid role")

	ErrNotFound          = errors.New("user: not found")
	ErrAlreadyExists     = errors.New("user: username or email already in use")
	ErrInvalidCredential = errors.New("user: invalid credentials")
	ErrForbidden         = errors.New("user: caller is not permitted to perform this action")
)
