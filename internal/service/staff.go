package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agnos-assignment/internal/model"
	"agnos-assignment/internal/repository"
)

var (
	ErrInvalidStaffInput  = errors.New("invalid staff input")
	ErrInvalidCredentials = errors.New("invalid login credentials")
	ErrHospitalNotFound   = repository.ErrHospitalNotFound
	ErrStaffAlreadyExists = repository.ErrStaffAlreadyExists
)

type HospitalFinder interface {
	FindByCode(context.Context, string) (model.Hospital, error)
}

type StaffStore interface {
	Create(context.Context, model.Staff) (model.Staff, error)
	FindByUsername(context.Context, int64, string) (model.Staff, error)
}

type PasswordManager interface {
	Hash(string) (string, error)
	Matches(string, string) bool
}

type TokenIssuer interface {
	Generate(model.Staff, model.Hospital) (string, time.Duration, error)
}

type StaffService struct {
	hospitals HospitalFinder
	staff     StaffStore
	passwords PasswordManager
	tokens    TokenIssuer
}

type CreateStaffInput struct {
	Username     string
	Password     string
	HospitalCode string
}

type CreatedStaff struct {
	ID           int64
	Username     string
	HospitalCode string
	CreatedAt    time.Time
}

type LoginStaffInput struct {
	Username     string
	Password     string
	HospitalCode string
}

type LoginResult struct {
	AccessToken string
	ExpiresIn   int64
}

func NewStaffService(
	hospitals HospitalFinder,
	staff StaffStore,
	passwords PasswordManager,
	tokens TokenIssuer,
) *StaffService {
	return &StaffService{
		hospitals: hospitals,
		staff:     staff,
		passwords: passwords,
		tokens:    tokens,
	}
}

func (s *StaffService) Login(ctx context.Context, input LoginStaffInput) (LoginResult, error) {
	username := strings.TrimSpace(input.Username)
	hospitalCode := strings.ToLower(strings.TrimSpace(input.HospitalCode))
	if username == "" || input.Password == "" || hospitalCode == "" || len(input.Password) > 72 {
		return LoginResult{}, ErrInvalidCredentials
	}

	hospital, err := s.hospitals.FindByCode(ctx, hospitalCode)
	if errors.Is(err, repository.ErrHospitalNotFound) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}

	staff, err := s.staff.FindByUsername(ctx, hospital.ID, username)
	if errors.Is(err, repository.ErrStaffNotFound) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}

	if !s.passwords.Matches(input.Password, staff.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}

	accessToken, expiresIn, err := s.tokens.Generate(staff, hospital)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return LoginResult{
		AccessToken: accessToken,
		ExpiresIn:   int64(expiresIn.Seconds()),
	}, nil
}

func (s *StaffService) CreateStaff(ctx context.Context, input CreateStaffInput) (CreatedStaff, error) {
	username := strings.TrimSpace(input.Username)
	hospitalCode := strings.ToLower(strings.TrimSpace(input.HospitalCode))
	if username == "" || hospitalCode == "" || len(input.Password) < 8 || len(input.Password) > 72 {
		return CreatedStaff{}, ErrInvalidStaffInput
	}

	hospital, err := s.hospitals.FindByCode(ctx, hospitalCode)
	if err != nil {
		return CreatedStaff{}, err
	}

	passwordHash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return CreatedStaff{}, fmt.Errorf("hash staff password: %w", err)
	}

	created, err := s.staff.Create(ctx, model.Staff{
		HospitalID:   hospital.ID,
		Username:     username,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return CreatedStaff{}, err
	}

	return CreatedStaff{
		ID:           created.ID,
		Username:     created.Username,
		HospitalCode: hospital.Code,
		CreatedAt:    created.CreatedAt,
	}, nil
}
