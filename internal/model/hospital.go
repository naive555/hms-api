package model

import "time"

type Hospital struct {
	ID         int64
	Code       string
	Name       string
	HISBaseURL string
	CreatedAt  time.Time
}
