package his

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agnos-assignment/internal/model"
)

const maxResponseBodyBytes = 1 << 20

type HospitalAClient struct {
	httpClient *http.Client
}

type hospitalAPatient struct {
	FirstNameTH  *string `json:"first_name_th"`
	MiddleNameTH *string `json:"middle_name_th"`
	LastNameTH   *string `json:"last_name_th"`
	FirstNameEN  *string `json:"first_name_en"`
	MiddleNameEN *string `json:"middle_name_en"`
	LastNameEN   *string `json:"last_name_en"`
	DateOfBirth  *string `json:"date_of_birth"`
	PatientHN    string  `json:"patient_hn"`
	NationalID   *string `json:"national_id"`
	PassportID   *string `json:"passport_id"`
	PhoneNumber  *string `json:"phone_number"`
	Email        *string `json:"email"`
	Gender       *string `json:"gender"`
}

func NewHospitalAClient(httpClient *http.Client) *HospitalAClient {
	return &HospitalAClient{httpClient: httpClient}
}

func (c *HospitalAClient) SearchByID(
	ctx context.Context,
	hospital model.Hospital,
	id string,
) (model.Patient, error) {
	if hospital.APIBaseURL == nil || strings.TrimSpace(*hospital.APIBaseURL) == "" {
		return model.Patient{}, errors.New("hospital API base URL is missing")
	}

	endpoint := strings.TrimRight(*hospital.APIBaseURL, "/") + "/patient/search/" + url.PathEscape(id)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return model.Patient{}, fmt.Errorf("create hospital request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return model.Patient{}, fmt.Errorf("send hospital request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return model.Patient{}, ErrPatientNotFound
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return model.Patient{}, fmt.Errorf("unexpected hospital status: %d", response.StatusCode)
	}

	var upstream hospitalAPatient
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBodyBytes))
	if err := decoder.Decode(&upstream); err != nil {
		return model.Patient{}, fmt.Errorf("decode hospital response: %w", err)
	}
	if upstream.PatientHN == "" {
		return model.Patient{}, errors.New("hospital response is missing patient_hn")
	}

	patient := model.Patient{
		PatientHN:    upstream.PatientHN,
		NationalID:   upstream.NationalID,
		PassportID:   upstream.PassportID,
		FirstNameTH:  upstream.FirstNameTH,
		MiddleNameTH: upstream.MiddleNameTH,
		LastNameTH:   upstream.LastNameTH,
		FirstNameEN:  upstream.FirstNameEN,
		MiddleNameEN: upstream.MiddleNameEN,
		LastNameEN:   upstream.LastNameEN,
		PhoneNumber:  upstream.PhoneNumber,
		Email:        upstream.Email,
	}
	if upstream.DateOfBirth != nil {
		dateOfBirth, err := time.Parse(time.DateOnly, *upstream.DateOfBirth)
		if err != nil {
			return model.Patient{}, fmt.Errorf("parse hospital date_of_birth: %w", err)
		}
		patient.DateOfBirth = &dateOfBirth
	}
	if upstream.Gender != nil {
		gender := model.Gender(*upstream.Gender)
		if gender != model.GenderMale && gender != model.GenderFemale {
			return model.Patient{}, fmt.Errorf("unsupported hospital gender: %q", *upstream.Gender)
		}
		patient.Gender = &gender
	}

	return patient, nil
}
