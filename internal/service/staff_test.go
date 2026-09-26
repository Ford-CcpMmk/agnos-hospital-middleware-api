package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"agnos-assignment/internal/model"
	"agnos-assignment/internal/repository"
	"agnos-assignment/internal/service"
)

type hospitalFinderFunc func(context.Context, string) (model.Hospital, error)

func (fn hospitalFinderFunc) FindByCode(ctx context.Context, code string) (model.Hospital, error) {
	return fn(ctx, code)
}

type staffStoreStub struct {
	createFn func(context.Context, model.Staff) (model.Staff, error)
	findFn   func(context.Context, int64, string) (model.Staff, error)
}

func (stub staffStoreStub) Create(ctx context.Context, staff model.Staff) (model.Staff, error) {
	return stub.createFn(ctx, staff)
}

func (stub staffStoreStub) FindByUsername(ctx context.Context, hospitalID int64, username string) (model.Staff, error) {
	return stub.findFn(ctx, hospitalID, username)
}

type passwordManagerStub struct {
	hashFn    func(string) (string, error)
	matchesFn func(string, string) bool
}

func (stub passwordManagerStub) Hash(password string) (string, error) {
	return stub.hashFn(password)
}

func (stub passwordManagerStub) Matches(password, encodedHash string) bool {
	return stub.matchesFn(password, encodedHash)
}

type tokenIssuerFunc func(model.Staff, model.Hospital) (string, time.Duration, error)

func (fn tokenIssuerFunc) Generate(staff model.Staff, hospital model.Hospital) (string, time.Duration, error) {
	return fn(staff, hospital)
}

func TestStaffServiceCreateStaff(t *testing.T) {
	createdAt := time.Date(2026, time.September, 25, 16, 0, 0, 0, time.UTC)
	var inserted model.Staff

	staffService := service.NewStaffService(
		hospitalFinderFunc(func(_ context.Context, code string) (model.Hospital, error) {
			if code != "hospital-a" {
				t.Fatalf("expected normalized hospital code, got %q", code)
			}
			return model.Hospital{ID: 7, Code: code}, nil
		}),
		staffStoreStub{
			createFn: func(_ context.Context, staff model.Staff) (model.Staff, error) {
				inserted = staff
				staff.ID = 12
				staff.CreatedAt = createdAt
				return staff, nil
			},
		},
		passwordManagerStub{hashFn: func(password string) (string, error) {
			if password != "password123" {
				t.Fatalf("expected original password to be hashed, got %q", password)
			}
			return "hashed-password", nil
		}},
		nil,
	)

	created, err := staffService.CreateStaff(context.Background(), service.CreateStaffInput{
		Username:     " staff01 ",
		Password:     "password123",
		HospitalCode: " HOSPITAL-A ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if inserted.HospitalID != 7 {
		t.Fatalf("expected hospital ID 7, got %d", inserted.HospitalID)
	}
	if inserted.Username != "staff01" {
		t.Fatalf("expected normalized username, got %q", inserted.Username)
	}
	if inserted.PasswordHash != "hashed-password" {
		t.Fatalf("expected hashed password, got %q", inserted.PasswordHash)
	}
	if created.ID != 12 || created.HospitalCode != "hospital-a" || !created.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected created staff: %+v", created)
	}
}

func TestStaffServiceCreateStaffErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         service.CreateStaffInput
		hospitalError error
		staffError    error
		expectedError error
	}{
		{
			name:          "invalid input",
			input:         service.CreateStaffInput{Username: "staff01", Password: "short", HospitalCode: "hospital-a"},
			expectedError: service.ErrInvalidStaffInput,
		},
		{
			name:          "hospital not found",
			input:         validCreateStaffInput(),
			hospitalError: repository.ErrHospitalNotFound,
			expectedError: service.ErrHospitalNotFound,
		},
		{
			name:          "duplicate staff",
			input:         validCreateStaffInput(),
			staffError:    repository.ErrStaffAlreadyExists,
			expectedError: service.ErrStaffAlreadyExists,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			staffService := service.NewStaffService(
				hospitalFinderFunc(func(context.Context, string) (model.Hospital, error) {
					return model.Hospital{ID: 7, Code: "hospital-a"}, test.hospitalError
				}),
				staffStoreStub{createFn: func(_ context.Context, staff model.Staff) (model.Staff, error) {
					return staff, test.staffError
				}},
				passwordManagerStub{hashFn: func(string) (string, error) {
					return "hashed-password", nil
				}},
				nil,
			)

			_, err := staffService.CreateStaff(context.Background(), test.input)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("expected error %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestStaffServiceLogin(t *testing.T) {
	staffService := service.NewStaffService(
		hospitalFinderFunc(func(_ context.Context, code string) (model.Hospital, error) {
			if code != "hospital-a" {
				t.Fatalf("expected normalized hospital code, got %q", code)
			}
			return model.Hospital{ID: 7, Code: code}, nil
		}),
		staffStoreStub{findFn: func(_ context.Context, hospitalID int64, username string) (model.Staff, error) {
			if hospitalID != 7 || username != "staff01" {
				t.Fatalf("unexpected staff lookup: hospital=%d username=%q", hospitalID, username)
			}
			return model.Staff{ID: 12, HospitalID: 7, Username: username, PasswordHash: "stored-hash"}, nil
		}},
		passwordManagerStub{matchesFn: func(password, encodedHash string) bool {
			return password == "password123" && encodedHash == "stored-hash"
		}},
		tokenIssuerFunc(func(staff model.Staff, hospital model.Hospital) (string, time.Duration, error) {
			if staff.ID != 12 || hospital.ID != 7 {
				t.Fatalf("unexpected token subjects: staff=%d hospital=%d", staff.ID, hospital.ID)
			}
			return "signed-token", time.Hour, nil
		}),
	)

	result, err := staffService.Login(context.Background(), service.LoginStaffInput{
		Username:     " staff01 ",
		Password:     "password123",
		HospitalCode: " HOSPITAL-A ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken != "signed-token" || result.ExpiresIn != 3600 {
		t.Fatalf("unexpected login result: %+v", result)
	}
}

func TestStaffServiceLoginRejectsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name          string
		input         service.LoginStaffInput
		hospitalError error
		staffError    error
		passwordMatch bool
	}{
		{name: "empty input", input: service.LoginStaffInput{}},
		{name: "unknown hospital", input: validLoginInput(), hospitalError: repository.ErrHospitalNotFound},
		{name: "unknown staff", input: validLoginInput(), staffError: repository.ErrStaffNotFound},
		{name: "wrong password", input: validLoginInput(), passwordMatch: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			staffService := service.NewStaffService(
				hospitalFinderFunc(func(context.Context, string) (model.Hospital, error) {
					return model.Hospital{ID: 7, Code: "hospital-a"}, test.hospitalError
				}),
				staffStoreStub{findFn: func(context.Context, int64, string) (model.Staff, error) {
					return model.Staff{ID: 12, HospitalID: 7, PasswordHash: "stored-hash"}, test.staffError
				}},
				passwordManagerStub{matchesFn: func(string, string) bool { return test.passwordMatch }},
				tokenIssuerFunc(func(model.Staff, model.Hospital) (string, time.Duration, error) {
					return "unexpected-token", time.Hour, nil
				}),
			)

			_, err := staffService.Login(context.Background(), test.input)
			if !errors.Is(err, service.ErrInvalidCredentials) {
				t.Fatalf("expected invalid credentials, got %v", err)
			}
		})
	}
}

func validCreateStaffInput() service.CreateStaffInput {
	return service.CreateStaffInput{
		Username:     "staff01",
		Password:     "password123",
		HospitalCode: "hospital-a",
	}
}

func validLoginInput() service.LoginStaffInput {
	return service.LoginStaffInput{
		Username:     "staff01",
		Password:     "password123",
		HospitalCode: "hospital-a",
	}
}
