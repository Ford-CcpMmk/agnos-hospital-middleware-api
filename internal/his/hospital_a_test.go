package his_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"agnos-assignment/internal/his"
	"agnos-assignment/internal/model"
)

func TestHospitalAClientSearchByID(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/patient/search/1103700123456" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
			"first_name_th":"สมชาย",
			"first_name_en":"Somchai",
			"last_name_en":"Jaidee",
			"date_of_birth":"1990-01-15",
			"patient_hn":"HN001",
			"national_id":"1103700123456",
			"gender":"M"
		}`)),
		}, nil
	})}

	client := his.NewHospitalAClient(httpClient)
	patient, err := client.SearchByID(
		context.Background(),
		model.Hospital{Code: "hospital-a", APIBaseURL: stringPointer("https://hospital-a.example")},
		"1103700123456",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if patient.PatientHN != "HN001" || patient.FirstNameEN == nil || *patient.FirstNameEN != "Somchai" {
		t.Fatalf("unexpected patient: %+v", patient)
	}
	if patient.DateOfBirth == nil || patient.DateOfBirth.Format("2006-01-02") != "1990-01-15" {
		t.Fatalf("unexpected date of birth: %v", patient.DateOfBirth)
	}
}

func TestHospitalAClientReturnsNotFound(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	client := his.NewHospitalAClient(httpClient)
	_, err := client.SearchByID(
		context.Background(),
		model.Hospital{Code: "hospital-a", APIBaseURL: stringPointer("https://hospital-a.example")},
		"missing",
	)
	if !errors.Is(err, his.ErrPatientNotFound) {
		t.Fatalf("expected patient-not-found error, got %v", err)
	}
}

func stringPointer(value string) *string {
	return &value
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
