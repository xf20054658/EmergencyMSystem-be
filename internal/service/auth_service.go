package service

import (
	"context"
	"fmt"
	"time"

	"emergency-msystem-backend/config"
	"emergency-msystem-backend/internal/database"
	pkgjwt "emergency-msystem-backend/pkg/jwt"

	"github.com/google/uuid"
)

// AuthService 认证服务
type AuthService struct {
	cfg config.Config
}

// NewAuthService 创建认证服务
func NewAuthService(cfg config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// LoginResult 登录结果
type LoginResult struct {
	Token       string `json:"token"`
	UserID      string `json:"user_id"`
	OpenID      string `json:"openid"`
	PhoneMasked string `json:"phone_masked"`
	IsNewUser   bool   `json:"is_new_user"`
	Role        string `json:"role"`
	VolunteerID string `json:"volunteer_id,omitempty"`
}

// WeChatLogin 微信登录
func (s *AuthService) WeChatLogin(ctx context.Context, code string) (*LoginResult, error) {
	// TODO: 调用微信 API 获取 openid
	// 这里使用模拟数据
	openID := "wx_" + uuid.New().String()[:16]
	return s.loginOrRegister(ctx, openID)
}

// PhoneLogin 手机号验证码登录
func (s *AuthService) PhoneLogin(ctx context.Context, phone, code string) (*LoginResult, error) {
	// TODO: 验证验证码
	_ = code

	// 查找或创建用户
	var userID, openID string
	var phoneMasked *string
	var isNew bool

	// 先按脱敏手机号查找
	err := database.Pool.QueryRow(ctx, `
		SELECT id, openid, phone_masked FROM users WHERE phone_masked = $1 LIMIT 1
	`, phone[:3]+"****"+phone[7:]).Scan(&userID, &openID, &phoneMasked)

	if err != nil {
		// 用户不存在，创建新用户
		isNew = true
		userID = uuid.New().String()
		openID = "mp_" + uuid.New().String()[:16]
		masked := phone[:3] + "****" + phone[7:]

		_, err := database.Pool.Exec(ctx, `
			INSERT INTO users (id, openid, phone_masked, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'active', NOW(), NOW())
		`, userID, openID, masked)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}

		// 游客合并：查找 guest_phone 匹配的求助记录
		_, _ = database.Pool.Exec(ctx, `
			UPDATE help_requests SET user_id = $1, guest_phone = NULL
			WHERE guest_phone = $2 AND user_id IS NULL
		`, userID, phone)
	}

	var resultPhoneMasked string
	if phoneMasked != nil {
		resultPhoneMasked = *phoneMasked
	}

	// 检查是否是志愿者
	volunteerID, role := s.getVolunteerInfo(ctx, userID)

	// 生成 JWT
	token, err := pkgjwt.GenerateToken(s.cfg.JWT, userID, openID, role, volunteerID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 更新登录时间
	_, _ = database.Pool.Exec(ctx, `
		UPDATE users SET last_login_at = NOW() WHERE id = $1
	`, userID)

	return &LoginResult{
		Token:       token,
		UserID:      userID,
		OpenID:      openID,
		PhoneMasked: resultPhoneMasked,
		IsNewUser:   isNew,
		Role:        role,
		VolunteerID: volunteerID,
	}, nil
}

// loginOrRegister 微信登录或注册
func (s *AuthService) loginOrRegister(ctx context.Context, openID string) (*LoginResult, error) {
	var userID string
	var isNew bool

	err := database.Pool.QueryRow(ctx, `
		SELECT id FROM users WHERE openid = $1 LIMIT 1
	`, openID).Scan(&userID)

	if err != nil {
		// 新用户
		isNew = true
		userID = uuid.New().String()
		_, err := database.Pool.Exec(ctx, `
			INSERT INTO users (id, openid, status, created_at, updated_at)
			VALUES ($1, $2, 'active', NOW(), NOW())
		`, userID, openID)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	// 获取志愿者信息
	volunteerID, role := s.getVolunteerInfo(ctx, userID)

	// 生成 JWT
	token, err := pkgjwt.GenerateToken(s.cfg.JWT, userID, openID, role, volunteerID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 更新登录时间
	_, _ = database.Pool.Exec(ctx, `
		UPDATE users SET last_login_at = NOW() WHERE id = $1
	`, userID)

	return &LoginResult{
		Token:       token,
		UserID:      userID,
		OpenID:      openID,
		IsNewUser:   isNew,
		Role:        role,
		VolunteerID: volunteerID,
	}, nil
}

func (s *AuthService) getVolunteerInfo(ctx context.Context, userID string) (volunteerID, role string) {
	err := database.Pool.QueryRow(ctx, `
		SELECT id FROM volunteers WHERE user_id = $1 LIMIT 1
	`, userID).Scan(&volunteerID)

	if err == nil && volunteerID != "" {
		return volunteerID, "volunteer"
	}
	return "", "user"
}

// GetUserProfile 获取用户信息（通过 mock 方式，开发阶段使用）
func (s *AuthService) GetUserProfile(ctx context.Context, userID string) (map[string]interface{}, error) {
	row := database.Pool.QueryRow(ctx, `
		SELECT id, openid, nickname, avatar_url, phone_masked, real_name, status,
			last_known_lat, last_known_lng, created_at
		FROM users WHERE id = $1
	`, userID)

	var (
		id, openID, status string
		nickname, avatarURL, phoneMasked, realName *string
		lastLat, lastLng *float64
		createdAt time.Time
	)

	err := row.Scan(&id, &openID, &nickname, &avatarURL, &phoneMasked, &realName,
		&status, &lastLat, &lastLng, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 获取志愿者信息
	volunteerID, _ := s.getVolunteerInfo(ctx, userID)

	result := map[string]interface{}{
		"id":          id,
		"openid":      openID,
		"status":      status,
		"created_at":  createdAt,
		"volunteer_id": volunteerID,
	}

	if nickname != nil { result["nickname"] = *nickname }
	if avatarURL != nil { result["avatar_url"] = *avatarURL }
	if phoneMasked != nil { result["phone_masked"] = *phoneMasked }
	if realName != nil { result["real_name"] = *realName }
	if lastLat != nil { result["last_known_lat"] = *lastLat }
	if lastLng != nil { result["last_known_lng"] = *lastLng }

	return result, nil
}

// UpdateLocation 更新用户最后位置
func (s *AuthService) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE users SET last_known_lat = $1, last_known_lng = $2, last_location_at = NOW()
		WHERE id = $3
	`, lat, lng, userID)
	return err
}
