package dto

import "time"

// --- Notification DTOs ---

// SubscribeRequest 订阅授权（兼容单数和复数字段名）
type SubscribeRequest struct {
	TemplateID  string   `json:"template_id"`
	TemplateIDs []string `json:"template_ids"`
	Type        string   `json:"subscribe_type,omitempty"`
	// 兼容旧格式
	SubscribeType string `json:"type,omitempty"`
}

// GetTemplateIDs 获取模板 ID 列表（支持单数和复数两种格式）
func (r *SubscribeRequest) GetTemplateIDs() []string {
	if len(r.TemplateIDs) > 0 {
		return r.TemplateIDs
	}
	if r.TemplateID != "" {
		return []string{r.TemplateID}
	}
	return nil
}

// GetSubscribeType 获取订阅类型
func (r *SubscribeRequest) GetSubscribeType() string {
	if r.SubscribeType != "" {
		return r.SubscribeType
	}
	return r.Type
}

// UpdatePreferenceRequest 更新通知偏好
type UpdatePreferenceRequest struct {
	TemplateID string `json:"template_id" binding:"required"`
	Enabled    bool   `json:"enabled"`
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
