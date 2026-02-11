package authentication

import (
	"context"
	"fmt"
	"time"

	"github.com/ajscimone/censys-challenge/internal/db"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int32  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type Authenticator struct {
	queries   *db.Queries
	jwtSecret []byte
}

func NewAuthenticator(queries *db.Queries, jwtSecret string) *Authenticator {
	return &Authenticator{
		queries:   queries,
		jwtSecret: []byte(jwtSecret),
	}
}

func (a *Authenticator) Login(ctx context.Context, email string) (string, error) {
	user, err := a.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (a *Authenticator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}