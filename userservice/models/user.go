// Package models holds the domain structs for the user service.
package models

import "time"

// User is an authenticated account. PasswordHash is a bcrypt hash and is never
// serialized to clients.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
