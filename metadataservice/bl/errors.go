// Package bl defines errors for metadata service business logic
package bl

import "errors"

var (
	// ErrInvalidTitle is returned when video title is empty
	ErrInvalidTitle = errors.New("video title cannot be empty")

	// ErrInvalidSize is returned when video size is invalid
	ErrInvalidSize = errors.New("video size must be greater than 0")

	// ErrInvalidVideoID is returned when video ID is invalid
	ErrInvalidVideoID = errors.New("invalid video ID")
)
