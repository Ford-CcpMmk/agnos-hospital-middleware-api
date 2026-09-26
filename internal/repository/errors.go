package repository

import "errors"

var (
	ErrHospitalNotFound   = errors.New("hospital not found")
	ErrStaffNotFound      = errors.New("staff not found")
	ErrStaffAlreadyExists = errors.New("staff already exists")
)
