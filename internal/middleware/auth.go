package middleware

import (
	"net/http"
	"strings"

	"emergency-msystem-backend/config"
	pkgjwt "emergency-msystem-backend/pkg/jwt"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthRequired JWT 鉴权中间件
func AuthRequired(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		authHeader = strings.TrimSpace(authHeader)
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "invalid authorization format")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			response.Unauthorized(c, "empty token")
			c.Abort()
			return
		}

		claims, err := pkgjwt.ParseToken(cfg, tokenString)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// 注入用户信息到上下文
		c.Set("user_id", claims.UserID)
		c.Set("openid", claims.OpenID)
		c.Set("role", claims.Role)
		c.Set("volunteer_id", claims.VolunteerID)

		c.Next()
	}
}

// OptionalAuth 可选鉴权（不强制要求token）
func OptionalAuth(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			claims, err := pkgjwt.ParseToken(cfg, parts[1])
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("openid", claims.OpenID)
				c.Set("role", claims.Role)
				c.Set("volunteer_id", claims.VolunteerID)
			}
		}
		c.Next()
	}
}

// VolunteerRequired 志愿者身份校验
func VolunteerRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "volunteer" && role != "admin" {
			response.Forbidden(c, "volunteer role required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminRequired 管理员身份校验
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			response.Forbidden(c, "admin role required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORSMiddleware CORS 中间件
func CORSMiddleware(cfg config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigin := "*"
		for _, o := range cfg.AllowedOrigins {
			if o == "*" || o == origin {
				allowedOrigin = origin
				break
			}
		}

		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
