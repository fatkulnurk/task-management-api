package jwt

import (
	"context"
	applicationtoken "taskmanagement/internal/application/token"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func New(secret string) applicationtoken.TokenService {
	return &TokenService{
		[]byte(secret),
		15 * time.Minute,
	}
}

func (tokenService *TokenService) Issue(_ context.Context, userID string) (string, int64, error) {
	expiration := time.Now().Add(tokenService.ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": expiration.Unix(),
	})
	rawToken, err := token.SignedString(tokenService.secret)
	return rawToken, int64(tokenService.ttl.Seconds()), err
}

func (tokenService *TokenService) Verify(_ context.Context, rawToken string) (string, error) {
	claimsToken, err := jwt.Parse(rawToken, func(_ *jwt.Token) (any, error) {
		return tokenService.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !claimsToken.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}
	userID, err := claimsToken.Claims.GetSubject()
	return userID, err
}
