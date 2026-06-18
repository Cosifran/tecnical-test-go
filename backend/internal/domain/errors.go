// Package domain defines sentinel (predefined) errors for the domain layer.
//
// WHY sentinel errors instead of fmt.Errorf everywhere: Sentinel errors let
// callers check WHAT went wrong using errors.Is(). For example:
//
//	err := userRepo.FindByEmail(ctx, "nonexistent@example.com")
//	if errors.Is(err, domain.ErrNotFound) {
//	    // handle "not found" case specifically
//	}
//
// This is more precise than string matching (fragile) and more idiomatic
// than custom error types (which require type assertion). For our use case,
// sentinel errors are the right level of abstraction.
package domain

import "errors"

// ErrNotFound means the requested resource does not exist in the database.
// For example: FindByEmail("nonexistent@example.com") returns ErrNotFound.
var ErrNotFound = errors.New("resource not found")

// ErrUnauthorized means the provided credentials are invalid or the
// authentication token is missing/expired/tampered.
// Used by: login with wrong password, invalid JWT, expired token.
var ErrUnauthorized = errors.New("unauthorized access")

// ErrValidation means the input data failed validation rules.
// For example: sensor data with an invalid type or a future timestamp.
var ErrValidation = errors.New("validation failed")

// ErrConflict means the operation conflicts with existing data.
// For example: creating a user with an email that already exists.
var ErrConflict = errors.New("resource conflict")

// ErrTokenExpired means the JWT token has passed its expiration time.
// This is a more specific version of ErrUnauthorized — we separate it
// because the spec requires different error responses:
//   - ErrTokenExpired → 401 with "token_expired" error code
//   - ErrUnauthorized → 401 with "invalid_token" error code
var ErrTokenExpired = errors.New("token has expired")

// ErrTokenInvalid means the JWT token is malformed or has an invalid signature.
// Separated from ErrTokenExpired for the same reason: different error codes.
var ErrTokenInvalid = errors.New("token is invalid")