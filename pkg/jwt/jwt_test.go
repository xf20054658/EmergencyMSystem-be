package jwt

import (
	"testing"
	"time"

	"emergency-msystem-backend/config"

	"github.com/stretchr/testify/assert"
)

func testConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:            "test-secret-key-for-unit-tests",
		AccessTokenExpire: 24 * time.Hour,
	}
}

func TestGenerateToken(t *testing.T) {
	cfg := testConfig()
	token, err := GenerateToken(cfg, "u_28", "openid_123", "user", "")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateToken_DifferentRoles(t *testing.T) {
	cfg := testConfig()

	t.Run("user token", func(t *testing.T) {
		token, err := GenerateToken(cfg, "u_001", "openid_001", "user", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("volunteer token", func(t *testing.T) {
		token, err := GenerateToken(cfg, "u_002", "openid_002", "volunteer", "v_001")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("admin token", func(t *testing.T) {
		token, err := GenerateToken(cfg, "u_003", "openid_003", "admin", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestParseToken_Valid(t *testing.T) {
	cfg := testConfig()
	tokenString, err := GenerateToken(cfg, "u_28", "openid_123", "user", "")
	assert.NoError(t, err)

	claims, err := ParseToken(cfg, tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "u_28", claims.UserID)
	assert.Equal(t, "openid_123", claims.OpenID)
	assert.Equal(t, "user", claims.Role)
}

func TestParseToken_Invalid(t *testing.T) {
	cfg := testConfig()

	t.Run("invalid signature", func(t *testing.T) {
		claims, err := ParseToken(cfg, "invalid.token.here")
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Equal(t, ErrTokenInvalid, err)
	})

	t.Run("tampered token", func(t *testing.T) {
		tokenString, _ := GenerateToken(cfg, "u_28", "openid_123", "user", "")
		tampered := tokenString[:len(tokenString)-5] + "XXXXX"
		claims, err := ParseToken(cfg, tampered)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("wrong secret", func(t *testing.T) {
		tokenString, _ := GenerateToken(cfg, "u_28", "openid_123", "user", "")
		wrongCfg := config.JWTConfig{
			Secret: "different-secret-key",
		}
		claims, err := ParseToken(wrongCfg, tokenString)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestExpiredToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:            "test-secret-for-expiry",
		AccessTokenExpire: -1 * time.Hour, // expired 1 hour ago
	}

	tokenString, err := GenerateToken(cfg, "u_expired", "openid_expired", "user", "")
	assert.NoError(t, err)

	claims, err := ParseToken(cfg, tokenString)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrTokenExpired, err)
}

func TestClaims_ContainsVolunteerID(t *testing.T) {
	cfg := testConfig()
	tokenString, err := GenerateToken(cfg, "u_vol", "openid_vol", "volunteer", "v_001")
	assert.NoError(t, err)

	claims, err := ParseToken(cfg, tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "v_001", claims.VolunteerID)
}

func TestClaims_RoleBasedAccess(t *testing.T) {
	t.Run("admin claims has admin role", func(t *testing.T) {
		cfg := testConfig()
		tokenString, err := GenerateToken(cfg, "u_admin", "oid_admin", "admin", "")
		assert.NoError(t, err)

		claims, _ := ParseToken(cfg, tokenString)
		assert.Equal(t, "admin", claims.Role)
	})

	t.Run("anonymous token", func(t *testing.T) {
		cfg := testConfig()
		now := time.Now()
		claims := Claims{
			UserID: "u_anon",
			Role:   "user",
			RegisteredClaims: RegisteredClaims{
				Issuer:    "emergency-msystem",
				Subject:   "u_anon",
				IssuedAt:  NewNumericDate(now),
				ExpiresAt: NewNumericDate(now.Add(cfg.AccessTokenExpire)),
			},
		}
		assert.Equal(t, "user", claims.Role)
		genToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := genToken.SignedString([]byte(cfg.Secret))
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)
		parsed, err := ParseToken(cfg, tokenString)
		assert.NoError(t, err)
		assert.Equal(t, "u_anon", parsed.UserID)
		assert.Equal(t, "user", parsed.Role)
	})
}
