package dto

// --- Volunteer DTOs ---

// CreateVolunteerRequest 注册志愿者
type CreateVolunteerRequest struct {
	DisplayName string   `json:"display_name" binding:"required,max=100"`
	Bio         string   `json:"bio,omitempty"`
	CertNo      string   `json:"cert_no,omitempty"`
	AreaCode    string   `json:"area_code" binding:"required"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	Skills      []VolunteerSkillItem `json:"skills,omitempty"`
}

// VolunteerSkillItem 技能项
type VolunteerSkillItem struct {
	SkillName   string `json:"skill_name" binding:"required"`
	Proficiency string `json:"proficiency"`
}

// UpdateVolunteerRequest 更新志愿者信息
type UpdateVolunteerRequest struct {
	DisplayName *string  `json:"display_name,omitempty"`
	Bio         *string  `json:"bio,omitempty"`
	AreaCode    *string  `json:"area_code,omitempty"`
	Lat         *float64 `json:"lat,omitempty"`
	Lng         *float64 `json:"lng,omitempty"`
	Status      *string  `json:"status,omitempty"`
}

// VolunteerListQuery 志愿者列表查询
type VolunteerListQuery struct {
	Tier     string `form:"tier"`
	Status   string `form:"status"`
	AreaCode string `form:"area_code"`
	Search   string `form:"search"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// VolunteerResponse 志愿者响应
type VolunteerResponse struct {
	ID             string   `json:"id"`
	UserID         string   `json:"user_id"`
	DisplayName    string   `json:"display_name"`
	Tier           string   `json:"tier"`
	TrustScore     int      `json:"trust_score"`
	Status         string   `json:"status"`
	Bio            string   `json:"bio,omitempty"`
	IDVerified     bool     `json:"id_verified"`
	CertNo         string   `json:"cert_no,omitempty"`
	AreaCode       string   `json:"area_code,omitempty"`
	AreaName       string   `json:"area_name,omitempty"`
	Lat            float64  `json:"lat,omitempty"`
	Lng            float64  `json:"lng,omitempty"`
	CompletedTasks int      `json:"completed_tasks"`
	Rating         float64  `json:"rating"`
	AbandonCount   int      `json:"abandon_count"`
	TotalAccepted  int      `json:"total_accepted"`
	FrozenUntil    string   `json:"frozen_until,omitempty"`
	Skills         []VolunteerSkillItem `json:"skills,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

// GPSReportRequest GPS位置上报
type GPSReportRequest struct {
	MatchID    string  `json:"match_id" binding:"required"`
	Lat        float64 `json:"lat" binding:"required"`
	Lng        float64 `json:"lng" binding:"required"`
	Accuracy   *float64 `json:"accuracy,omitempty"`
	Speed      *float64 `json:"speed,omitempty"`
	Heading    *float64 `json:"heading,omitempty"`
}
