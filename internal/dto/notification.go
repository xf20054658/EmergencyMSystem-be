package dto

import "time"

// --- Notification DTOs ---

// SubscribeRequest 订阅授权
type SubscribeRequest struct {
	TemplateID string `json:"template_id" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=longterm once"`
}

// UpdatePreferenceRequest 更新通知偏好
type UpdatePreferenceRequest struct {
	TemplateID string `json:"template_id" binding:"required"`
	Enabled    bool   `json:"enabled" binding:"required"`
}

// NotificationListQuery 通知列表查询
type NotificationListQuery struct {
	ReadStatus *bool `form:"read_status"`
	Urgency    string `form:"urgency"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// NotificationResponse 通知响应
type NotificationResponse struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Urgency    string    `json:"urgency"`
	ActionType string    `json:"action_type,omitempty"`
	ActionData interface{} `json:"action_data,omitempty"`
	ReadStatus bool      `json:"read_status"`
	CreatedAt  time.Time `json:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
}

// PreferenceResponse 偏好响应
type PreferenceResponse struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	Enabled      bool   `json:"enabled"`
}
