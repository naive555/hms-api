package model

import "time"

type Patient struct {
	ID           int64     `json:"-"`
	HospitalID   int64     `json:"-"`
	PatientHN    string    `json:"patient_hn"`
	FirstNameTH  *string   `json:"first_name_th"`
	MiddleNameTH *string   `json:"middle_name_th"`
	LastNameTH   *string   `json:"last_name_th"`
	FirstNameEN  *string   `json:"first_name_en"`
	MiddleNameEN *string   `json:"middle_name_en"`
	LastNameEN   *string   `json:"last_name_en"`
	DateOfBirth  *Date     `json:"date_of_birth"`
	NationalID   *string   `json:"national_id"`
	PassportID   *string   `json:"passport_id"`
	PhoneNumber  *string   `json:"phone_number"`
	Email        *string   `json:"email"`
	Gender       *string   `json:"gender"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}
