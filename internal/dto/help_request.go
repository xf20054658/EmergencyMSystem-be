package dto

// --- HelpRequest DTOs ---

// CreateHelpRequest 创建求助
type CreateHelpRequest struct {
	DisasterEventID  *string  `json:"disaster_event_id"`
	AreaCode         string   `json:"area_code" binding:"required"`
	Address          string   `json:"address" binding:"required"`
	Lat              *float64 `json:"lat"`
	Lng              *float64 `json:"lng"`
	Content          string   `json:"content" binding:"required"`
	Urgency          string   `json:"urgency" binding:"required,oneof=critical high medium low"`
	PeopleCount      int      `json:"people_count"`
	WaterLevel       *string  `json:"water_level,omitempty"`
	Images           []string `json:"images,omitempty"`
	IsProxy          bool     `json:"is_proxy"`
	ProxyPhone       *string  `json:"proxy_phone,omitempty"`
	ProxyName        *string  `json:"proxy_name,omitempty"`
	ProxyNotifyMethod string  `json:"proxy_notify_method"`
	VulnerableGroups []string `json:"vulnerable_groups,omitempty"`
	// 物资需求
	Needs []RequestNeedItem `json:"needs,omitempty"`
}

// RequestNeedItem 物资需求项
type RequestNeedItem struct {
	NeedName string `json:"need_name" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
	Unit     string `json:"unit"`
}

// UpdateHelpRequest 更新求助
type UpdateHelpRequest struct {
	Address     *string   `json:"address,omitempty"`
	Content     *string   `json:"content,omitempty"`
	Urgency     *string   `json:"urgency,omitempty"`
	PeopleCount *int      `json:"people_count,omitempty"`
	Status      *string   `json:"status,omitempty"`
	WaterLevel  *string   `json:"water_level,omitempty"`
	Images      []string  `json:"images,omitempty"`
}

// HelpRequestListQuery 求助列表查询
type HelpRequestListQuery struct {
	Status    string `form:"status"`
	Urgency   string `form:"urgency"`
	Route     string `form:"route"`
	AreaCode  string `form:"area_code"`
	UserID    string `form:"user_id"`
	Search    string `form:"search"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

// HelpRequestResponse 求助响应
type HelpRequestResponse struct {
	ID               string   `json:"id"`
	RequestNo        string   `json:"request_no"`
	UserID           string   `json:"user_id,omitempty"`
	Nickname         string   `json:"nickname,omitempty"`
	PhoneMasked      string   `json:"phone_masked,omitempty"`
	DisasterEventID  string   `json:"disaster_event_id,omitempty"`
	AreaCode         string   `json:"area_code"`
	AreaName         string   `json:"area_name,omitempty"`
	Address          string   `json:"address"`
	Lat              float64  `json:"lat,omitempty"`
	Lng              float64  `json:"lng,omitempty"`
	Content          string   `json:"content"`
	Urgency          string   `json:"urgency"`
	PeopleCount      int      `json:"people_count"`
	Status           string   `json:"status"`
	Route            string   `json:"route"`
	WaterLevel       string   `json:"water_level,omitempty"`
	Images           []string `json:"images,omitempty"`
	// v2.1 新增
	IsProxy          bool     `json:"is_proxy"`
	ProxyPhone       string   `json:"proxy_phone,omitempty"`
	ProxyName        string   `json:"proxy_name,omitempty"`
	ProxyNotifyMethod string  `json:"proxy_notify_method,omitempty"`
	VulnerableGroups []string `json:"vulnerable_groups,omitempty"`
	// 关联数据
	Needs    []RequestNeedItem `json:"needs,omitempty"`
	Matches  []MatchBrief      `json:"matches,omitempty"`
	// 时间
	ResolvedAt string `json:"resolved_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// MatchBrief 匹配简要信息
type MatchBrief struct {
	ID            string `json:"id"`
	MatchNo       string `json:"match_no"`
	Status        string `json:"status"`
	VolunteerName string `json:"volunteer_name,omitempty"`
	VolunteerTier string `json:"volunteer_tier,omitempty"`
	DistanceKM    float64 `json:"distance_km,omitempty"`
}
