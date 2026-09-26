package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agnos-assignment/internal/his"
	"agnos-assignment/internal/model"
	"agnos-assignment/internal/repository"
)

const (
	defaultPageSize = 20
	maximumPageSize = 100
)

var (
	ErrNoSearchCriteria     = errors.New("at least one search criterion is required")
	ErrInvalidSearchInput   = errors.New("invalid patient search input")
	ErrInvalidHospitalScope = errors.New("invalid hospital scope")
	ErrHISUnavailable       = errors.New("hospital information system unavailable")
)

type PatientHospitalFinder interface {
	FindByID(context.Context, int64) (model.Hospital, error)
}

type PatientStore interface {
	Search(context.Context, repository.PatientSearchFilter) ([]model.Patient, int64, error)
	Upsert(context.Context, model.Patient) (model.Patient, error)
}

type HISPatientFinder interface {
	SearchByID(context.Context, model.Hospital, string) (model.Patient, error)
}

type PatientService struct {
	hospitals PatientHospitalFinder
	patients  PatientStore
	his       HISPatientFinder
}

type SearchPatientsInput struct {
	HospitalID   int64
	HospitalCode string
	NationalID   *string
	PassportID   *string
	FirstName    *string
	MiddleName   *string
	LastName     *string
	DateOfBirth  *string
	PhoneNumber  *string
	Email        *string
	Page         int
	PageSize     int
}

type SearchPatientsResult struct {
	Patients []model.Patient
	Page     int
	PageSize int
	Total    int64
}

func NewPatientService(
	hospitals PatientHospitalFinder,
	patients PatientStore,
	his HISPatientFinder,
) *PatientService {
	return &PatientService{hospitals: hospitals, patients: patients, his: his}
}

func (s *PatientService) Search(
	ctx context.Context,
	input SearchPatientsInput,
) (SearchPatientsResult, error) {
	normalizeSearchInput(&input)
	if !hasSearchCriteria(input) {
		return SearchPatientsResult{}, ErrNoSearchCriteria
	}

	dateOfBirth, err := parseOptionalDate(input.DateOfBirth)
	if err != nil {
		return SearchPatientsResult{}, ErrInvalidSearchInput
	}

	page, pageSize := normalizePagination(input.Page, input.PageSize)
	filter := repository.PatientSearchFilter{
		HospitalID:  input.HospitalID,
		NationalID:  input.NationalID,
		PassportID:  input.PassportID,
		FirstName:   input.FirstName,
		MiddleName:  input.MiddleName,
		LastName:    input.LastName,
		DateOfBirth: dateOfBirth,
		PhoneNumber: input.PhoneNumber,
		Email:       input.Email,
		Limit:       pageSize,
		Offset:      (page - 1) * pageSize,
	}

	patients, total, err := s.patients.Search(ctx, filter)
	if err != nil {
		return SearchPatientsResult{}, err
	}
	if len(patients) > 0 || (input.NationalID == nil && input.PassportID == nil) {
		return SearchPatientsResult{Patients: patients, Page: page, PageSize: pageSize, Total: total}, nil
	}

	hospital, err := s.hospitals.FindByID(ctx, input.HospitalID)
	if errors.Is(err, repository.ErrHospitalNotFound) || hospital.Code != input.HospitalCode {
		return SearchPatientsResult{}, ErrInvalidHospitalScope
	}
	if err != nil {
		return SearchPatientsResult{}, err
	}

	identifier := input.NationalID
	if identifier == nil {
		identifier = input.PassportID
	}
	upstreamPatient, err := s.his.SearchByID(ctx, hospital, *identifier)
	if errors.Is(err, his.ErrPatientNotFound) {
		return SearchPatientsResult{Patients: []model.Patient{}, Page: page, PageSize: pageSize, Total: 0}, nil
	}
	if err != nil {
		return SearchPatientsResult{}, fmt.Errorf("%w: %v", ErrHISUnavailable, err)
	}

	upstreamPatient.HospitalID = hospital.ID
	if !patientMatches(upstreamPatient, input, dateOfBirth) {
		return SearchPatientsResult{Patients: []model.Patient{}, Page: page, PageSize: pageSize, Total: 0}, nil
	}

	saved, err := s.patients.Upsert(ctx, upstreamPatient)
	if err != nil {
		return SearchPatientsResult{}, err
	}

	return SearchPatientsResult{
		Patients: []model.Patient{saved},
		Page:     page,
		PageSize: pageSize,
		Total:    1,
	}, nil
}

func normalizeSearchInput(input *SearchPatientsInput) {
	input.HospitalCode = strings.ToLower(strings.TrimSpace(input.HospitalCode))
	input.NationalID = normalizeOptionalString(input.NationalID)
	input.PassportID = normalizeOptionalString(input.PassportID)
	input.FirstName = normalizeOptionalString(input.FirstName)
	input.MiddleName = normalizeOptionalString(input.MiddleName)
	input.LastName = normalizeOptionalString(input.LastName)
	input.DateOfBirth = normalizeOptionalString(input.DateOfBirth)
	input.PhoneNumber = normalizeOptionalString(input.PhoneNumber)
	input.Email = normalizeOptionalString(input.Email)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func hasSearchCriteria(input SearchPatientsInput) bool {
	return input.NationalID != nil || input.PassportID != nil || input.FirstName != nil ||
		input.MiddleName != nil || input.LastName != nil || input.DateOfBirth != nil ||
		input.PhoneNumber != nil || input.Email != nil
}

func parseOptionalDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, *value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maximumPageSize {
		pageSize = maximumPageSize
	}
	return page, pageSize
}

func patientMatches(patient model.Patient, input SearchPatientsInput, dateOfBirth *time.Time) bool {
	return exactMatch(patient.NationalID, input.NationalID) &&
		exactMatch(patient.PassportID, input.PassportID) &&
		nameMatch(patient.FirstNameTH, patient.FirstNameEN, input.FirstName) &&
		nameMatch(patient.MiddleNameTH, patient.MiddleNameEN, input.MiddleName) &&
		nameMatch(patient.LastNameTH, patient.LastNameEN, input.LastName) &&
		dateMatch(patient.DateOfBirth, dateOfBirth) &&
		exactMatch(patient.PhoneNumber, input.PhoneNumber) &&
		exactMatch(patient.Email, input.Email)
}

func exactMatch(actual, expected *string) bool {
	return expected == nil || (actual != nil && *actual == *expected)
}

func nameMatch(thaiName, englishName, expected *string) bool {
	if expected == nil {
		return true
	}
	wanted := strings.ToLower(*expected)
	return (thaiName != nil && strings.Contains(strings.ToLower(*thaiName), wanted)) ||
		(englishName != nil && strings.Contains(strings.ToLower(*englishName), wanted))
}

func dateMatch(actual, expected *time.Time) bool {
	return expected == nil || (actual != nil && actual.Equal(*expected))
}
