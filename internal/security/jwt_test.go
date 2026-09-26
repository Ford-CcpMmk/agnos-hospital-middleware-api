package security_test

import (
	"errors"
	"testing"
	"time"

	"agnos-assignment/internal/model"
	"agnos-assignment/internal/security"
)

func TestJWTManagerGenerateAndParse(t *testing.T) {
	manager := security.NewJWTManager("test-secret", time.Hour)

	token, ttl, err := manager.Generate(
		model.Staff{ID: 12, HospitalID: 7},
		model.Hospital{ID: 7, Code: "hospital-a"},
	)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if ttl != time.Hour {
		t.Fatalf("expected one-hour TTL, got %s", ttl)
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.StaffID != 12 || claims.HospitalID != 7 || claims.HospitalCode != "hospital-a" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTManagerRejectsInvalidTokens(t *testing.T) {
	issuer := security.NewJWTManager("issuer-secret", time.Hour)
	token, _, err := issuer.Generate(
		model.Staff{ID: 12, HospitalID: 7},
		model.Hospital{ID: 7, Code: "hospital-a"},
	)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	tests := []struct {
		name    string
		manager *security.JWTManager
		token   string
	}{
		{name: "malformed", manager: issuer, token: "not-a-jwt"},
		{name: "wrong signature", manager: security.NewJWTManager("different-secret", time.Hour), token: token},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.manager.Parse(test.token)
			if !errors.Is(err, security.ErrInvalidToken) {
				t.Fatalf("expected invalid token, got %v", err)
			}
		})
	}
}
