package dto

// --- Auth DTOs ---

// LoginRequest 微信登录请求
type LoginRequest struct {
	Code string `json:"code" binding:"required"`
}

// PhoneLoginRequest 手机号登录
type PhoneLoginRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
	Code  string `json:"code" binding:"required,len=6"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token       string `json:"token"`
	UserID      string `json:"user_id"`
	OpenID      string `json:"openid"`
	Nickname    string `json:"nickname,omitempty"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	IsNewUser   bool   `json:"is_new_user"`
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
