package router_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agnos-assignment/internal/model"
	"agnos-assignment/internal/router"
	"agnos-assignment/internal/security"
	"agnos-assignment/internal/service"

	"github.com/gin-gonic/gin"
)

type databaseStub struct {
	err error
}

func (stub databaseStub) Ping(context.Context) error {
	return stub.err
}

type staffServiceStub struct {
	created     service.CreatedStaff
	createError error
	loginResult service.LoginResult
	loginError  error
}

func (stub staffServiceStub) CreateStaff(context.Context, service.CreateStaffInput) (service.CreatedStaff, error) {
	return stub.created, stub.createError
}

func (stub staffServiceStub) Login(context.Context, service.LoginStaffInput) (service.LoginResult, error) {
	return stub.loginResult, stub.loginError
}

type patientServiceStub struct {
	result service.SearchPatientsResult
	err    error
}

func (stub patientServiceStub) Search(context.Context, service.SearchPatientsInput) (service.SearchPatientsResult, error) {
	return stub.result, stub.err
}

type tokenParserStub struct {
	claims security.Claims
	err    error
}

func (stub tokenParserStub) Parse(string) (security.Claims, error) {
	return stub.claims, stub.err
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router.New(databaseStub{}, staffServiceStub{}, patientServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	const expectedBody = `{"status":"ok"}`
	if response.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, response.Body.String())
	}
}

func TestReady(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		database     databaseStub
		expectedCode int
	}{
		{name: "database available", database: databaseStub{}, expectedCode: http.StatusOK},
		{name: "database unavailable", database: databaseStub{err: errors.New("database unavailable")}, expectedCode: http.StatusServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			response := httptest.NewRecorder()

			router.New(test.database, staffServiceStub{}, patientServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

			if response.Code != test.expectedCode {
				t.Fatalf("expected status %d, got %d", test.expectedCode, response.Code)
			}
		})
	}
}

func TestCreateStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, time.September, 25, 16, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		body         string
		staff        staffServiceStub
		expectedCode int
	}{
		{
			name: "created",
			body: `{"username":"staff01","password":"password123","hospital":"hospital-a"}`,
			staff: staffServiceStub{created: service.CreatedStaff{
				ID: 1, Username: "staff01", HospitalCode: "hospital-a", CreatedAt: createdAt,
			}},
			expectedCode: http.StatusCreated,
		},
		{
			name:         "invalid request",
			body:         `{"username":"staff01"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "unknown field",
			body:         `{"username":"staff01","password":"password123","hospital":"hospital-a","role":"admin"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "duplicate staff",
			body:         `{"username":"staff01","password":"password123","hospital":"hospital-a"}`,
			staff:        staffServiceStub{createError: service.ErrStaffAlreadyExists},
			expectedCode: http.StatusConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/staff/create",
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			router.New(databaseStub{}, test.staff, patientServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

			if response.Code != test.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", test.expectedCode, response.Code, response.Body.String())
			}
		})
	}
}

func TestLoginStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		body         string
		staff        staffServiceStub
		expectedCode int
	}{
		{
			name:         "logged in",
			body:         `{"username":"staff01","password":"password123","hospital":"hospital-a"}`,
			staff:        staffServiceStub{loginResult: service.LoginResult{AccessToken: "signed-token", ExpiresIn: 3600}},
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid request",
			body:         `{"username":"staff01"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid credentials",
			body:         `{"username":"staff01","password":"wrong-password","hospital":"hospital-a"}`,
			staff:        staffServiceStub{loginError: service.ErrInvalidCredentials},
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/staff/login",
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			router.New(databaseStub{}, test.staff, patientServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

			if response.Code != test.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", test.expectedCode, response.Code, response.Body.String())
			}
		})
	}
}

func TestSearchPatients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	firstName := "Somchai"

	tests := []struct {
		name          string
		authorization string
		body          string
		patients      patientServiceStub
		tokens        tokenParserStub
		expectedCode  int
	}{
		{
			name:          "search completed",
			authorization: "Bearer valid-token",
			body:          `{"first_name":"Somchai"}`,
			patients: patientServiceStub{result: service.SearchPatientsResult{
				Patients: []model.Patient{{PatientHN: "HN001", FirstNameEN: &firstName}},
				Page:     1, PageSize: 20, Total: 1,
			}},
			tokens: tokenParserStub{claims: security.Claims{
				StaffID: 12, HospitalID: 7, HospitalCode: "hospital-a",
			}},
			expectedCode: http.StatusOK,
		},
		{
			name:         "missing token",
			body:         `{"first_name":"Somchai"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:          "empty criteria",
			authorization: "Bearer valid-token",
			body:          `{}`,
			patients:      patientServiceStub{err: service.ErrNoSearchCriteria},
			tokens: tokenParserStub{claims: security.Claims{
				StaffID: 12, HospitalID: 7, HospitalCode: "hospital-a",
			}},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/patient/search",
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", "application/json")
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			response := httptest.NewRecorder()

			router.New(databaseStub{}, staffServiceStub{}, test.patients, test.tokens).ServeHTTP(response, request)

			if response.Code != test.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", test.expectedCode, response.Code, response.Body.String())
			}
		})
	}
}
