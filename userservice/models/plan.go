package models

import "time"

// Plan is a subscription tier. AmountMinor is the price in the smallest unit of
// Currency (e.g. paise for INR). PeriodDays is the access window granted when a
// payment for this plan is captured.
type Plan struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	PeriodDays  int       `json:"period_days"`
	CreatedAt   time.Time `json:"created_at"`
}
