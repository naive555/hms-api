package model

import "time"

type Staff struct {
	ID           int64
	HospitalID   int64
	Username     string
	PasswordHash string `json:"-"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
