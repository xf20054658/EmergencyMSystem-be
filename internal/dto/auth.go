package dto

// --- Auth DTOs ---

// LoginRequest 登录请求（支持微信code、手机号+密码、手机号+OTP）
type LoginRequest struct {
	Code     string `json:"code,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Password string `json:"password,omitempty"`
	OtpCode  string `json:"otp_code,omitempty"`
}

// PhoneLoginRequest 手机号验证码登录
type PhoneLoginRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
	Code  string `json:"code" binding:"required,len=6"`
}

// WxLoginRequest 微信小程序登录请求
type WxLoginRequest struct {
	JsCode        string `json:"js_code" binding:"required"`
	EncryptedData string `json:"encryptedData" binding:"required"`
	Iv            string `json:"iv" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token      string    `json:"token"`
	ExpiresAt  string    `json:"expires_at"`
	User       UserBrief `json:"user"`
	IsNewUser  bool      `json:"is_new_user"`
}

// UserBrief 前端需要的用户简要信息
type UserBrief struct {
	ID          string `json:"id"`
	Phone       string `json:"phone"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Status      string `json:"status"`
	VolunteerID string `json:"volunteer_id,omitempty"`
}

// LoginResult 内部登录结果
type LoginResult struct {
	Token       string
	UserID      string
	OpenID      string
	PhoneMasked string
	Phone       string
	Nickname    string
	AvatarURL   string
	IsNewUser   bool
	Role        string
	VolunteerID string
	ExpiresAt   string
}

// SendOTPRequest 发送验证码
type SendOTPRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
}

// RegisterRequest 注册（绑定手机号）
type RegisterRequest struct {
	Phone    string `json:"phone" binding:"required,len=11"`
	Code     string `json:"code" binding:"required,len=6"`
	Nickname string `json:"nickname" binding:"required,max=100"`
	RealName string `json:"real_name,omitempty"`
	IDCard   string `json:"id_card,omitempty"`
}

// UserProfileResponse 用户信息
type UserProfileResponse struct {
	ID           string  `json:"id"`
	OpenID       string  `json:"openid"`
	Nickname     string  `json:"nickname,omitempty"`
	AvatarURL    string  `json:"avatar_url,omitempty"`
	PhoneMasked  string  `json:"phone_masked,omitempty"`
	RealName     string  `json:"real_name,omitempty"`
	Status       string  `json:"status"`
	LastKnownLat float64 `json:"last_known_lat,omitempty"`
	LastKnownLng float64 `json:"last_known_lng,omitempty"`
}

// UpdateLocationRequest 更新定位
type UpdateLocationRequest struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}
