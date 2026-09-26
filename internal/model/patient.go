package model

import "time"

type Gender string

const (
	GenderMale   Gender = "M"
	GenderFemale Gender = "F"
)

type Patient struct {
	ID           int64
	HospitalID   int64
	PatientHN    string
	NationalID   *string
	PassportID   *string
	FirstNameTH  *string
	MiddleNameTH *string
	LastNameTH   *string
	FirstNameEN  *string
	MiddleNameEN *string
	LastNameEN   *string
	DateOfBirth  *time.Time
	PhoneNumber  *string
	Email        *string
	Gender       *Gender
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
