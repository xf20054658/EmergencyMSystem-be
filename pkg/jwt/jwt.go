package jwt

import (
	"errors"
	"time"

	"emergency-msystem-backend/config"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserID   string `json:"user_id"`
	OpenID   string `json:"openid"`
	Role     string `json:"role"` // user / volunteer / admin
	VolunteerID string `json:"volunteer_id,omitempty"`
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)

// GenerateToken 生成 JWT
func GenerateToken(cfg config.JWTConfig, userID, openID, role, volunteerID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:      userID,
		OpenID:      openID,
		Role:        role,
		VolunteerID: volunteerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "emergency-msystem",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTokenExpire)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 解析 JWT
func ParseToken(cfg config.JWTConfig, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
