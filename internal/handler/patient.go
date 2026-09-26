package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"agnos-assignment/internal/dto"
	"agnos-assignment/internal/middleware"
	"agnos-assignment/internal/model"
	"agnos-assignment/internal/service"

	"github.com/gin-gonic/gin"
)

type PatientSearcher interface {
	Search(context.Context, service.SearchPatientsInput) (service.SearchPatientsResult, error)
}

type PatientHandler struct {
	patients PatientSearcher
}

func NewPatientHandler(patients PatientSearcher) *PatientHandler {
	return &PatientHandler{patients: patients}
}

func (h *PatientHandler) Search(c *gin.Context) {
	claims, ok := middleware.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "authentication is required"))
		return
	}

	var request dto.SearchPatientsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "invalid patient search input"))
		return
	}

	result, err := h.patients.Search(c.Request.Context(), service.SearchPatientsInput{
		HospitalID:   claims.HospitalID,
		HospitalCode: claims.HospitalCode,
		NationalID:   request.NationalID,
		PassportID:   request.PassportID,
		FirstName:    request.FirstName,
		MiddleName:   request.MiddleName,
		LastName:     request.LastName,
		DateOfBirth:  request.DateOfBirth,
		PhoneNumber:  request.PhoneNumber,
		Email:        request.Email,
		Page:         request.Page,
		PageSize:     request.PageSize,
	})
	if err != nil {
		h.handleSearchError(c, err)
		return
	}

	patients := make([]dto.PatientData, 0, len(result.Patients))
	for _, patient := range result.Patients {
		patients = append(patients, patientData(patient))
	}

	c.JSON(http.StatusOK, dto.SearchPatientsResponse{
		Data: patients,
		Pagination: dto.Pagination{
			Page:     result.Page,
			PageSize: result.PageSize,
			Total:    result.Total,
		},
	})
}

func (h *PatientHandler) handleSearchError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNoSearchCriteria), errors.Is(err, service.ErrInvalidSearchInput):
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", err.Error()))
	case errors.Is(err, service.ErrInvalidHospitalScope):
		c.JSON(http.StatusForbidden, dto.NewErrorResponse("FORBIDDEN", "invalid hospital scope"))
	case errors.Is(err, service.ErrHISUnavailable):
		c.JSON(http.StatusBadGateway, dto.NewErrorResponse("HIS_UNAVAILABLE", "hospital information system is unavailable"))
	default:
		slog.Error("patient search failed", "error", err)
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("INTERNAL_ERROR", "internal server error"))
	}
}

func patientData(patient model.Patient) dto.PatientData {
	var dateOfBirth *string
	if patient.DateOfBirth != nil {
		formatted := patient.DateOfBirth.Format(time.DateOnly)
		dateOfBirth = &formatted
	}

	var gender *string
	if patient.Gender != nil {
		formatted := string(*patient.Gender)
		gender = &formatted
	}

	return dto.PatientData{
		FirstNameTH:  patient.FirstNameTH,
		MiddleNameTH: patient.MiddleNameTH,
		LastNameTH:   patient.LastNameTH,
		FirstNameEN:  patient.FirstNameEN,
		MiddleNameEN: patient.MiddleNameEN,
		LastNameEN:   patient.LastNameEN,
		DateOfBirth:  dateOfBirth,
		PatientHN:    patient.PatientHN,
		NationalID:   patient.NationalID,
		PassportID:   patient.PassportID,
		PhoneNumber:  patient.PhoneNumber,
		Email:        patient.Email,
		Gender:       gender,
	}
}
