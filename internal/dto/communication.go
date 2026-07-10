package dto

// --- Communication DTOs ---

// BindPhoneRequest AXB绑定请求
type BindPhoneRequest struct {
	Provider string `json:"provider" binding:"required,oneof=aliyun tencent"`
}

// CallRequest 发起通话
type CallRequest struct {
	CallType string `json:"call_type" binding:"required,oneof=voip phone"`
}

// CallRecordResponse 通话记录响应
type CallRecordResponse struct {
	ID           string `json:"id"`
	MatchID      string `json:"match_id"`
	CallType     string `json:"call_type"`
	CallerRole   string `json:"caller_role"`
	DurationSec  int    `json:"duration_sec"`
	Status       string `json:"status"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time,omitempty"`
}

// VirtualPhoneResponse 虚拟号码绑定响应
type VirtualPhoneResponse struct {
	ID           string `json:"id"`
	VirtualPhone string `json:"virtual_phone"`
	Status       string `json:"status"`
	ExpireTime   string `json:"expire_time"`
}
