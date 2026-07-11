package handler

import (
	"emergency-msystem-backend/config"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/internal/service"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	svc *service.AuthService
	cfg config.Config
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(svc *service.AuthService, cfg config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

// Login 统一登录（自动检测：密码登录 / 微信登录 / OTP登录）
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	// 密码登录：phone + password
	if req.Phone != "" && req.Password != "" {
		result, err := h.svc.PasswordLogin(c, req.Phone, req.Password)
		if err != nil {
			response.Unauthorized(c, err.Error())
			return
		}
		response.Success(c, result)
		return
	}

	// OTP 登录：phone + otp_code
	if req.Phone != "" && req.OtpCode != "" {
		result, err := h.svc.PhoneLogin(c, req.Phone, req.OtpCode)
		if err != nil {
			response.InternalError(c, "login failed: "+err.Error())
			return
		}
		response.Success(c, result)
		return
	}

	// 微信登录：code
	if req.Code != "" {
		result, err := h.svc.WeChatLogin(c, req.Code)
		if err != nil {
			response.InternalError(c, "login failed: "+err.Error())
			return
		}
		response.Success(c, result)
		return
	}

	response.BadRequest(c, "请提供 phone+password / phone+otp_code / code")
}

// PhoneLogin 手机号验证码登录
func (h *AuthHandler) PhoneLogin(c *gin.Context) {
	var req dto.PhoneLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	result, err := h.svc.PhoneLogin(c, req.Phone, req.Code)
	if err != nil {
		response.InternalError(c, "login failed: "+err.Error())
		return
	}

	response.Success(c, result)
}

// WxLogin 微信小程序登录（code2session + 解密手机号）
func (h *AuthHandler) WxLogin(c *gin.Context) {
	var req dto.WxLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	result, err := h.svc.WxMiniProgramLogin(c, req.JsCode, req.EncryptedData, req.Iv)
	if err != nil {
		response.InternalError(c, "wechat login failed: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetProfile 获取用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	profile, err := h.svc.GetUserProfile(c, userID.(string))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, profile)
}

// UpdateLocation 更新位置
func (h *AuthHandler) UpdateLocation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req dto.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	if err := h.svc.UpdateLocation(c, userID.(string), req.Lat, req.Lng); err != nil {
		response.InternalError(c, "update location failed")
		return
	}

	response.Success(c, nil)
}
