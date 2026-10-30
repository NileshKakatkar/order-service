package order

import (
	"crypto/rsa"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"strings"
)

type Claims struct {
	Role  string `json:"role"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func authenticate(path string) fiber.Handler {
	data, err := os.ReadFile(path)
	var key *rsa.PublicKey
	if err == nil {
		key, err = jwt.ParseRSAPublicKeyFromPEM(data)
	}
	return func(c *fiber.Ctx) error {
		if err != nil || key == nil {
			return fiber.NewError(500, "authentication configuration unavailable")
		}
		header := strings.TrimSpace(c.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			return fiber.NewError(401, "bearer token required")
		}
		claims := &Claims{}
		token, parseErr := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fiber.NewError(401, "invalid signing algorithm")
			}
			return key, nil
		}, jwt.WithIssuer("shopcore-user-service"), jwt.WithAudience("shopcore"))
		if parseErr != nil || token == nil || !token.Valid {
			return fiber.NewError(401, "invalid or expired token")
		}
		c.Locals("userID", claims.Subject)
		c.Locals("role", claims.Role)
		c.Locals("email", claims.Email)
		return c.Next()
	}
}
