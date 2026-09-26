package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"agnos-assignment/internal/dto"
	"agnos-assignment/internal/service"

	"github.com/gin-gonic/gin"
)

type StaffService interface {
	CreateStaff(context.Context, service.CreateStaffInput) (service.CreatedStaff, error)
	Login(context.Context, service.LoginStaffInput) (service.LoginResult, error)
}

type StaffHandler struct {
	staff StaffService
}

func NewStaffHandler(staff StaffService) *StaffHandler {
	return &StaffHandler{staff: staff}
}

func (h *StaffHandler) Login(c *gin.Context) {
	var request dto.LoginStaffRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"VALIDATION_ERROR",
			"username, password, and hospital are required",
		))
		return
	}

	result, err := h.staff.Login(c.Request.Context(), service.LoginStaffInput{
		Username:     request.Username,
		Password:     request.Password,
		HospitalCode: request.Hospital,
	})
	if errors.Is(err, service.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			"INVALID_CREDENTIALS",
			"invalid login credentials",
		))
		return
	}
	if err != nil {
		slog.Error("staff login failed", "error", err)
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("INTERNAL_ERROR", "internal server error"))
		return
	}

	c.JSON(http.StatusOK, dto.LoginStaffResponse{
		Data: dto.LoginStaffData{
			AccessToken: result.AccessToken,
			TokenType:   "Bearer",
			ExpiresIn:   result.ExpiresIn,
		},
	})
}

func (h *StaffHandler) Create(c *gin.Context) {
	var request dto.CreateStaffRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"VALIDATION_ERROR",
			"username, password, and hospital must be valid",
		))
		return
	}

	created, err := h.staff.CreateStaff(c.Request.Context(), service.CreateStaffInput{
		Username:     request.Username,
		Password:     request.Password,
		HospitalCode: request.Hospital,
	})
	if err != nil {
		h.handleCreateError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateStaffResponse{
		Data: dto.CreateStaffData{
			ID:        created.ID,
			Username:  created.Username,
			Hospital:  created.HospitalCode,
			CreatedAt: created.CreatedAt,
		},
	})
}

func (h *StaffHandler) handleCreateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidStaffInput):
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "invalid staff input"))
	case errors.Is(err, service.ErrHospitalNotFound):
		c.JSON(http.StatusNotFound, dto.NewErrorResponse("HOSPITAL_NOT_FOUND", "hospital not found"))
	case errors.Is(err, service.ErrStaffAlreadyExists):
		c.JSON(http.StatusConflict, dto.NewErrorResponse("STAFF_ALREADY_EXISTS", "staff already exists in this hospital"))
	default:
		slog.Error("create staff failed", "error", err)
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("INTERNAL_ERROR", "internal server error"))
	}
}
