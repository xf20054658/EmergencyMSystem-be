package dto

// --- Match DTOs ---

// MatchListQuery 匹配列表查询
type MatchListQuery struct {
	Status      string `form:"status"`
	RequestID   string `form:"request_id"`
	VolunteerID string `form:"volunteer_id"`
	IsCooperative *bool `form:"is_cooperative"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

// MatchResponse 匹配响应
type MatchResponse struct {
	ID                      string  `json:"id"`
	MatchNo                 string  `json:"match_no"`
	RequestID               string  `json:"request_id"`
	RequestNo               string  `json:"request_no,omitempty"`
	RequestAddress          string  `json:"request_address,omitempty"`
	RequestUrgency          string  `json:"request_urgency,omitempty"`
	VolunteerID             string  `json:"volunteer_id"`
	VolunteerName           string  `json:"volunteer_name,omitempty"`
	VolunteerTier           string  `json:"volunteer_tier,omitempty"`
	VolunteerTrustScore     int     `json:"volunteer_trust_score,omitempty"`
	Status                  string  `json:"status"`
	RiskLevel               string  `json:"risk_level"`
	DistanceKM              float64 `json:"distance_km,omitempty"`
	MatchRadiusUsed         float64 `json:"match_radius_used,omitempty"`
	ETAMin                  int     `json:"eta_min,omitempty"`
	TimeoutSec              int     `json:"timeout_sec"`
	ElapsedSec              int     `json:"elapsed_sec"`
	VirtualPhone            string  `json:"virtual_phone,omitempty"`
	// v2.1
	RelayHop               int     `json:"relay_hop"`
	RelayGroupID            string  `json:"relay_group_id,omitempty"`
	IsCooperative           bool    `json:"is_cooperative"`
	CooperativeGroupID      string  `json:"cooperative_group_id,omitempty"`
	WaitingReason           string  `json:"waiting_reason,omitempty"`
	WaitingETAMin           int     `json:"waiting_eta_min,omitempty"`
	UnreachableDetectedAt   string  `json:"unreachable_detected_at,omitempty"`
	LocationSharingEnabled  bool    `json:"location_sharing_enabled"`
	// 时间
	MatchTime    string `json:"match_time"`
	AcceptTime   string `json:"accept_time,omitempty"`
	ArriveTime   string `json:"arrive_time,omitempty"`
	CompleteTime string `json:"complete_time,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// AcceptMatchRequest 接受匹配
type AcceptMatchRequest struct {
	ETAMin *int `json:"eta_min"`
}

// CompleteMatchRequest 完成任务
type CompleteMatchRequest struct {
	SubmitPhotos []string `json:"submit_photos,omitempty"`
	SubmitNote   string   `json:"submit_note,omitempty"`
}

// SetWaitingRequest 设置等待状态
type SetWaitingRequest struct {
	Reason string `json:"reason" binding:"required,max=200"`
	ETAMin int    `json:"eta_min" binding:"required,min=1"`
}

// ReinforceRequest 增援请求
type ReinforceRequest struct {
	Reason      string `json:"reason" binding:"required,max=300"`
	NeededCount int    `json:"needed_count" binding:"required,min=1"`
}

// ConfirmCompletionRequest 确认完成
type ConfirmCompletionRequest struct {
	DisputeNote string `json:"dispute_note,omitempty"`
}

// MatchMonitorResponse 匹配监控响应（用于PC指挥中心）
type MatchMonitorResponse struct {
	AlertState string  `json:"alert_state"` // normal/warning/timeout_imminent/unreachable/waiting
	Matches    []MatchResponse `json:"matches"`
}
