package domain

import "errors"

var ErrNotFound = errors.New("resource not found")

var ErrUnauthorized = errors.New("unauthorized access")

var ErrValidation = errors.New("validation failed")

var ErrConflict = errors.New("resource conflict")

// ErrTokenExpired is separated from ErrUnauthorized because the API returns different error codes.
var ErrTokenExpired = errors.New("token has expired")

var ErrTokenInvalid = errors.New("token is invalid")