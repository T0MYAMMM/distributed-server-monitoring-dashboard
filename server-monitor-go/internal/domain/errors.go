package domain

import "errors"

// Sentinel errors returned by the service and storage layers. The HTTP error
// mapper (transport layer) translates these into status codes and a consistent
// JSON error envelope, so handlers never spell out status codes for them.
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrNotAllowed is returned when an agent reports under a name that is not
	// on the allow-list (the ingest 403 case).
	ErrNotAllowed = errors.New("not allowed")
	// ErrUnauthorized is returned when an admin action lacks valid credentials.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrConflict is returned when creating an entity that already exists.
	ErrConflict = errors.New("conflict")
	// ErrInvalidInput is returned for malformed or missing request data.
	ErrInvalidInput = errors.New("invalid input")
)
