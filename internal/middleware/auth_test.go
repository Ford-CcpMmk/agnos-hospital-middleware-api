package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agnos-assignment/internal/middleware"
	"agnos-assignment/internal/model"
	"agnos-assignment/internal/security"

	"github.com/gin-gonic/gin"
)

func TestAuthenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := security.NewJWTManager("test-secret", time.Hour)
	token, _, err := manager.Generate(
		model.Staff{ID: 12, HospitalID: 7},
		model.Hospital{ID: 7, Code: "hospital-a"},
	)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	tests := []struct {
		name          string
		authorization string
		expectedCode  int
	}{
		{name: "valid bearer token", authorization: "Bearer " + token, expectedCode: http.StatusOK},
		{name: "missing token", expectedCode: http.StatusUnauthorized},
		{name: "invalid token", authorization: "Bearer invalid", expectedCode: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/protected", middleware.Authenticate(manager), func(c *gin.Context) {
				claims, ok := middleware.ClaimsFromContext(c.Request.Context())
				if !ok || claims.StaffID != 12 || claims.HospitalID != 7 {
					t.Fatalf("expected authenticated claims, got %+v", claims)
				}
				c.Status(http.StatusOK)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.expectedCode {
				t.Fatalf("expected status %d, got %d", test.expectedCode, response.Code)
			}
		})
	}
}
