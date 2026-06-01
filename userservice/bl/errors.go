// Package bl holds the business logic for the user service: auth, subscriptions,
// payments, and the reconciliation/sweeper jobs.
package bl

import "errors"

var (
	// ErrEmailTaken is returned when registering an already-used email.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidCredentials is returned on a failed login.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrPlanNotFound is returned when subscribing to an unknown plan.
	ErrPlanNotFound = errors.New("plan not found")
	// ErrAlreadySubscribed is returned when an active subscription already exists.
	ErrAlreadySubscribed = errors.New("already subscribed to this plan")
	// ErrInvalidToken is returned when a refresh token is invalid or wrong-typed.
	ErrInvalidToken = errors.New("invalid token")
	// ErrInvalidSignature is returned when a webhook signature fails verification.
	ErrInvalidSignature = errors.New("invalid webhook signature")
)
