package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"emergency-msystem-backend/config"
	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	pkgjwt "emergency-msystem-backend/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
type AuthService struct {
	cfg config.Config
}

// NewAuthService 创建认证服务
func NewAuthService(cfg config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// Bcrypt cost factor
const bcryptCost = 12

// HashPassword 对明文密码进行 bcrypt 哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword 验证密码
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// convertResult 将 LoginResult 转为 LoginResponse
func convertResult(result dto.LoginResult) dto.LoginResponse {
	phone := result.Phone
	if phone == "" {
		phone = result.PhoneMasked
	}
	name := result.Nickname
	if name == "" {
		name = result.PhoneMasked
	}
	return dto.LoginResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		IsNewUser: result.IsNewUser,
		User: dto.UserBrief{
			ID:          result.UserID,
			Phone:       phone,
			PhoneMasked: result.PhoneMasked,
			Name:        name,
			Role:        result.Role,
			AvatarURL:   result.AvatarURL,
			Status:      "active",
			VolunteerID: result.VolunteerID,
		},
	}
}

// WxSessionResult 微信 code2Session 返回
type WxSessionResult struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

// WxPhoneData 微信手机号加密数据解密后的结构
type WxPhoneData struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
}

// WeChatLogin 微信PC登录（仅code → openid）
func (s *AuthService) WeChatLogin(ctx context.Context, code string) (*dto.LoginResponse, error) {
	// TODO: 调用微信 OAuth2 API 获取 openid
	openID := "wx_" + uuid.New().String()[:16]
	result, err := s.loginOrRegister(ctx, openID)
	if err != nil {
		return nil, err
	}
	resp := convertResult(*result)
	return &resp, nil
}

// WxMiniProgramLogin 微信小程序登录（code2session + 解密手机号）
func (s *AuthService) WxMiniProgramLogin(ctx context.Context, jsCode, encryptedData, iv string) (*dto.LoginResponse, error) {
	// 1. 调用微信 code2Session 获取 openid 和 session_key
	openID, sessionKey, err := s.code2Session(jsCode)
	if err != nil {
		return nil, fmt.Errorf("code2Session failed: %w", err)
	}

	// 2. 用 session_key 解密 encryptedData 获取手机号
	phone, err := decryptPhoneNumber(encryptedData, sessionKey, iv)
	if err != nil {
		return nil, fmt.Errorf("decrypt phone failed: %w", err)
	}

	// 3. 手机号脱敏
	masked := maskPhone(phone)

	// 4. 登录或注册
	result, err := s.loginOrRegisterWithPhone(ctx, openID, phone, masked)
	if err != nil {
		return nil, err
	}

	resp := convertResult(*result)
	return &resp, nil
}

// code2Session 调用微信 jscode2session 接口
func (s *AuthService) code2Session(jsCode string) (openID, sessionKey string, err error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.cfg.WeChat.AppID, s.cfg.WeChat.AppSecret, jsCode,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	var result WxSessionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", "", fmt.Errorf("wechat api error: %d %s", result.ErrCode, result.ErrMsg)
	}

	return result.OpenID, result.SessionKey, nil
}

// decryptPhoneNumber 用 session_key + iv 解密 encryptedData 得到手机号
func decryptPhoneNumber(encryptedData, sessionKey, iv string) (string, error) {
	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("base64 decode encryptedData: %w", err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return "", fmt.Errorf("base64 decode sessionKey: %w", err)
	}
	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return "", fmt.Errorf("base64 decode iv: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	decrypted := make([]byte, len(encryptedBytes))
	mode := cipher.NewCBCDecrypter(block, ivBytes)
	mode.CryptBlocks(decrypted, encryptedBytes)

	// PKCS7 unpad
	decrypted = pkcs7Unpad(decrypted)
	if decrypted == nil {
		return "", fmt.Errorf("pkcs7 unpad failed")
	}

	var phoneData WxPhoneData
	if err := json.Unmarshal(decrypted, &phoneData); err != nil {
		return "", fmt.Errorf("parse phone data: %w", err)
	}

	return phoneData.PurePhoneNumber, nil
}

// pkcs7Unpad 移除 PKCS7 填充
func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding > aes.BlockSize {
		return nil
	}
	return data[:len(data)-padding]
}

// maskPhone 手机号脱敏 13812345678 → 138****5678
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// loginOrRegisterWithPhone 微信登录或注册（同时更新手机号）
func (s *AuthService) loginOrRegisterWithPhone(ctx context.Context, openID, phone, masked string) (*dto.LoginResult, error) {
	var userID string
	var isNew bool

	err := database.Pool.QueryRow(ctx, `
		SELECT id FROM users WHERE openid = $1 LIMIT 1
	`, openID).Scan(&userID)

	if err != nil {
		// 新用户：创建
		isNew = true
		userID = uuid.New().String()
		_, err := database.Pool.Exec(ctx, `
			INSERT INTO users (id, openid, phone_masked, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'active', NOW(), NOW())
		`, userID, openID, masked)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	} else {
		// 已有用户：更新手机号
		_, _ = database.Pool.Exec(ctx, `
			UPDATE users SET phone_masked = $1, updated_at = NOW()
			WHERE id = $2 AND phone_masked IS NULL
		`, masked, userID)
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

	return &dto.LoginResult{
		Token:       token,
		UserID:      userID,
		OpenID:      openID,
		PhoneMasked: masked,
		Phone:       phone,
		IsNewUser:   isNew,
		Role:        role,
		VolunteerID: volunteerID,
		ExpiresAt:   time.Now().Add(s.cfg.JWT.AccessTokenExpire).Format(time.RFC3339),
	}, nil
}

// PasswordLogin 手机号 + 密码登录（PC 端）
func (s *AuthService) PasswordLogin(ctx context.Context, phone, password string) (*dto.LoginResponse, error) {
	var (
		userID       string
		openID       string
		nickname     *string
		avatarURL    *string
		phoneMasked  *string
		passwordHash *string
	)

	masked := phone[:3] + "****" + phone[7:]
	err := database.Pool.QueryRow(ctx, `
		SELECT id, openid, nickname, avatar_url, phone_masked, password_hash
		FROM users WHERE phone_masked = $1 AND password_hash IS NOT NULL
		LIMIT 1
	`, masked).Scan(&userID, &openID, &nickname, &avatarURL, &phoneMasked, &passwordHash)

	if err != nil {
		return nil, fmt.Errorf("账号或密码错误")
	}

	if passwordHash == nil || !VerifyPassword(*passwordHash, password) {
		return nil, fmt.Errorf("账号或密码错误")
	}

	// 检查是否是志愿者
	volunteerID, role := s.getVolunteerInfo(ctx, userID)

	// 如果没有志愿者的 admin/调度员角色，查 volunteer 表
	if role == "user" {
		role = s.getRoleInfo(ctx, userID)
	}

	// 生成 JWT
	token, err := pkgjwt.GenerateToken(s.cfg.JWT, userID, openID, role, volunteerID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 更新登录时间
	_, _ = database.Pool.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)

	var n, av, pm string
	if nickname != nil { n = *nickname }
	if avatarURL != nil { av = *avatarURL }
	if phoneMasked != nil { pm = *phoneMasked }

	resp := convertResult(dto.LoginResult{
		Token:       token,
		UserID:      userID,
		OpenID:      openID,
		Nickname:    n,
		AvatarURL:   av,
		PhoneMasked: pm,
		Phone:       phone,
		IsNewUser:   false,
		Role:        role,
		VolunteerID: volunteerID,
		ExpiresAt:   time.Now().Add(s.cfg.JWT.AccessTokenExpire).Format(time.RFC3339),
	})
	return &resp, nil
}

// PhoneLogin 手机号验证码登录
func (s *AuthService) PhoneLogin(ctx context.Context, phone, code string) (*dto.LoginResponse, error) {
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

	resp := convertResult(dto.LoginResult{
		Token:       token,
		UserID:      userID,
		OpenID:      openID,
		PhoneMasked: resultPhoneMasked,
		Phone:       phone,
		IsNewUser:   isNew,
		Role:        role,
		VolunteerID: volunteerID,
		ExpiresAt:   time.Now().Add(s.cfg.JWT.AccessTokenExpire).Format(time.RFC3339),
	})
	return &resp, nil
}

// loginOrRegister 微信登录或注册
func (s *AuthService) loginOrRegister(ctx context.Context, openID string) (*dto.LoginResult, error) {
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

	return &dto.LoginResult{
		Token:       token,
		UserID:      userID,
		OpenID:      openID,
		IsNewUser:   isNew,
		Role:        role,
		VolunteerID: volunteerID,
		ExpiresAt:   time.Now().Add(s.cfg.JWT.AccessTokenExpire).Format(time.RFC3339),
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

// getRoleInfo 通过手机号检查是否有管理员/调度员角色（password_hash 非空即为后台用户）
func (s *AuthService) getRoleInfo(ctx context.Context, userID string) string {
	var passwordHash *string
	err := database.Pool.QueryRow(ctx, `
		SELECT password_hash FROM users WHERE id = $1 AND password_hash IS NOT NULL
	`, userID).Scan(&passwordHash)
	if err == nil && passwordHash != nil {
		// 手机号以 138 开头且 admin 标记 — 简单判断
		return "admin"
	}
	return "user"
}

// GetUserProfile 获取用户信息
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
	volunteerID, role := s.getVolunteerInfo(ctx, userID)
	if role == "user" {
		role = s.getRoleInfo(ctx, userID)
	}

	result := map[string]interface{}{
		"id":           id,
		"openid":       openID,
		"status":       status,
		"role":         role,
		"created_at":   createdAt,
		"volunteer_id": volunteerID,
	}

	if nickname != nil { result["nickname"] = *nickname }
	if avatarURL != nil { result["avatar_url"] = *avatarURL }
	if phoneMasked != nil { result["phone_masked"] = *phoneMasked; result["phone"] = *phoneMasked }
	if realName != nil { result["real_name"] = *realName; result["name"] = *realName }
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
