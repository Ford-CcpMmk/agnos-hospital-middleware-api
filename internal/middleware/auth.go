package middleware

import (
	"context"
	"net/http"
	"strings"

	"agnos-assignment/internal/dto"
	"agnos-assignment/internal/security"

	"github.com/gin-gonic/gin"
)

type claimsContextKey struct{}

type TokenParser interface {
	Parse(string) (security.Claims, error)
}

func Authenticate(tokens TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		scheme, tokenString, found := strings.Cut(authorization, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(tokenString) == "" {
			unauthorized(c)
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(tokenString))
		if err != nil {
			unauthorized(c)
			return
		}

		ctx := context.WithValue(c.Request.Context(), claimsContextKey{}, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func ClaimsFromContext(ctx context.Context) (security.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(security.Claims)
	return claims, ok
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(
		http.StatusUnauthorized,
		dto.NewErrorResponse("UNAUTHORIZED", "a valid bearer token is required"),
	)
}
