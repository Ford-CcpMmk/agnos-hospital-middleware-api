package model

import "time"

type Hospital struct {
	ID         int64
	Code       string
	Name       string
	APIBaseURL *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
