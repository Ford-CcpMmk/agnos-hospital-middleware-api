package security

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"agnos-assignment/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

const tokenIssuer = "agnos-assignment"

var ErrInvalidToken = errors.New("invalid access token")

type Claims struct {
	StaffID      int64  `json:"staff_id"`
	HospitalID   int64  `json:"hospital_id"`
	HospitalCode string `json:"hospital_code"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m *JWTManager) Generate(staff model.Staff, hospital model.Hospital) (string, time.Duration, error) {
	now := time.Now().UTC()
	claims := Claims{
		StaffID:      staff.ID,
		HospitalID:   hospital.ID,
		HospitalCode: hospital.Code,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   strconv.FormatInt(staff.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	signedToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, m.ttl, nil
}

func (m *JWTManager) Parse(tokenString string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}
