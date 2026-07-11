package models

import "time"

// --- Enums (匹配 PostgreSQL 枚举类型) ---

type (
	DisasterType     string // flood/drought/fire/earthquake/mudslide/typhoon/other
	ResponseLevel    string // I/II/III/IV
	Urgency          string // critical/high/medium/low
	RequestStatus    string // pending/processing/resolved/cancelled
	Route            string // centralized/p2p
	TrustTier        string // tier1/tier2/tier3
	VolunteerStatus  string // active/available/offline
	MatchStatus      string // pending/accepted/enroute/waiting/unreachable/completed/rejected/timeout/escalated
	EscalationLevel  string // warning/danger
	EscalationAction string // pending/auto_escalated/manual_escalated/resolved
	BindingStatus    string // bound/in_call/expired/unbound
	CallType         string // voip/phone
	CallStatus       string // success/failed/no_answer/busy/in_progress
	CallerRole       string // requester/provider
	NotifType        string // longterm/once
	NotifUrgency     string // urgent/high/medium/info
	ShelterStatus    string // open/full/preparing/closed
	TeamType         string // fire/military/militia/ngo/police/medical/coast_guard
	TeamStatus       string // active/standby/rest
	AnnounceLevel    string // urgent/normal
	WaterLevel       string // none/light/moderate/severe
	VulnerableGroup  string // elderly/child/pregnant/disabled/patient/infant
	AreaDensity      string // urban/suburban/rural
	BatchStatus      string // preparing/in_transit/arrived/distributing/distributed/recalled/expired
	ResourceSource   string // government/donation/procurement
	AlertSource      string // national/provincial/manual
	AlertLevel       string // red/orange/yellow/blue
	CompletionStatus string // submitted/confirmed/auto_confirmed/disputed
	ReviewDirection  string // requester_to_provider/provider_to_requester
	SOSStatus        string // triggered/confirmed/cancelled/false_alarm
	LocationSource   string // gps/cell_tower/ip/last_known
	ProxyNotifyMethod string // sms/wechat/none
	ReinforceStatus  string // pending/accepted/rejected/cancelled
	TriggerAction    string // escalate_route/expand_radius/add_dispatcher/broadcast_alert
)

// --- Domain Models (33 tables) ---

// DisasterEvent 灾害事件
type DisasterEvent struct {
	ID             string       `json:"id" db:"id"`
	DisasterType   DisasterType `json:"disaster_type" db:"disaster_type"`
	Name           string       `json:"name" db:"name"`
	Level          ResponseLevel `json:"level" db:"level"`
	Status         string       `json:"status" db:"status"`
	StartTime      time.Time    `json:"start_time" db:"start_time"`
	EndTime        *time.Time   `json:"end_time,omitempty" db:"end_time"`
	EpicenterLat   *float64     `json:"epicenter_lat,omitempty" db:"epicenter_lat"`
	EpicenterLng   *float64     `json:"epicenter_lng,omitempty" db:"epicenter_lng"`
	Description    *string      `json:"description,omitempty" db:"description"`
	AffectedCities []string     `json:"affected_cities,omitempty" db:"affected_cities"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at" db:"updated_at"`
}

// Area 行政区域
type Area struct {
	ID         int        `json:"id" db:"id"`
	Code       string     `json:"code" db:"code"`
	Name       string     `json:"name" db:"name"`
	ParentCode *string    `json:"parent_code,omitempty" db:"parent_code"`
	Level      string     `json:"level" db:"level"`
	Lng        *float64   `json:"lng,omitempty" db:"lng"`
	Lat        *float64   `json:"lat,omitempty" db:"lat"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// User 用户
type User struct {
	ID             string     `json:"id" db:"id"`
	OpenID         string     `json:"openid" db:"openid"`
	UnionID        *string    `json:"unionid,omitempty" db:"unionid"`
	Nickname       *string    `json:"nickname,omitempty" db:"nickname"`
	AvatarURL      *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	PhoneEncrypted []byte     `json:"-" db:"phone_encrypted"`
	PhoneMasked    *string    `json:"phone_masked,omitempty" db:"phone_masked"`
	RealName       *string    `json:"real_name,omitempty" db:"real_name"`
	IDCardHash     *string    `json:"-" db:"id_card_hash"`
	PasswordHash   *string    `json:"-" db:"password_hash"`
	Status         string     `json:"status" db:"status"`
	AreaCode       *string    `json:"area_code,omitempty" db:"area_code"`
	GuestPhone     *string    `json:"guest_phone,omitempty" db:"guest_phone"`
	LastKnownLat   *float64   `json:"last_known_lat,omitempty" db:"last_known_lat"`
	LastKnownLng   *float64   `json:"last_known_lng,omitempty" db:"last_known_lng"`
	LastLocationAt *time.Time `json:"last_location_at,omitempty" db:"last_location_at"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Volunteer 志愿者
type Volunteer struct {
	ID             string          `json:"id" db:"id"`
	UserID         string          `json:"user_id" db:"user_id"`
	DisplayName    string          `json:"display_name" db:"display_name"`
	Tier           TrustTier       `json:"tier" db:"tier"`
	TrustScore     int             `json:"trust_score" db:"trust_score"`
	Status         VolunteerStatus `json:"status" db:"status"`
	Bio            *string         `json:"bio,omitempty" db:"bio"`
	IDVerified     bool            `json:"id_verified" db:"id_verified"`
	CertNo         *string         `json:"cert_no,omitempty" db:"cert_no"`
	CertExpire     *time.Time      `json:"cert_expire,omitempty" db:"cert_expire"`
	AreaCode       *string         `json:"area_code,omitempty" db:"area_code"`
	Lat            *float64        `json:"lat,omitempty" db:"lat"`
	Lng            *float64        `json:"lng,omitempty" db:"lng"`
	CompletedTasks int             `json:"completed_tasks" db:"completed_tasks"`
	Rating         float64         `json:"rating" db:"rating"`
	AbandonCount   int             `json:"abandon_count" db:"abandon_count"`
	TotalAccepted  int             `json:"total_accepted" db:"total_accepted"`
	FrozenUntil    *time.Time      `json:"frozen_until,omitempty" db:"frozen_until"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// VolunteerSkill 志愿者技能
type VolunteerSkill struct {
	ID          int       `json:"id" db:"id"`
	VolunteerID string    `json:"volunteer_id" db:"volunteer_id"`
	SkillName   string    `json:"skill_name" db:"skill_name"`
	Proficiency string    `json:"proficiency" db:"proficiency"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// HelpRequest 求助请求
type HelpRequest struct {
	ID               string             `json:"id" db:"id"`
	RequestNo        string             `json:"request_no" db:"request_no"`
	UserID           *string            `json:"user_id,omitempty" db:"user_id"`
	DisasterEventID  *string            `json:"disaster_event_id,omitempty" db:"disaster_event_id"`
	AreaCode         string             `json:"area_code" db:"area_code"`
	Address          string             `json:"address" db:"address"`
	Lat              *float64           `json:"lat,omitempty" db:"lat"`
	Lng              *float64           `json:"lng,omitempty" db:"lng"`
	Content          string             `json:"content" db:"content"`
	Urgency          Urgency            `json:"urgency" db:"urgency"`
	PeopleCount      int                `json:"people_count" db:"people_count"`
	Status           RequestStatus      `json:"status" db:"status"`
	Route            Route              `json:"route" db:"route"`
	WaterLevel       *WaterLevel        `json:"water_level,omitempty" db:"water_level"`
	Images           []string           `json:"images,omitempty" db:"images"`
	IsProxy          bool               `json:"is_proxy" db:"is_proxy"`
	ProxyPhone       *string            `json:"proxy_phone,omitempty" db:"proxy_phone"`
	ProxyName        *string            `json:"proxy_name,omitempty" db:"proxy_name"`
	ProxyNotifyMethod ProxyNotifyMethod `json:"proxy_notify_method" db:"proxy_notify_method"`
	ProxyNotifiedAt  *time.Time         `json:"proxy_notified_at,omitempty" db:"proxy_notified_at"`
	VulnerableGroups []VulnerableGroup  `json:"vulnerable_groups,omitempty" db:"vulnerable_groups"`
	ResolvedAt       *time.Time         `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" db:"updated_at"`
}

// RequestNeed 求助需求物资
type RequestNeed struct {
	ID        int       `json:"id" db:"id"`
	RequestID string    `json:"request_id" db:"request_id"`
	NeedName  string    `json:"need_name" db:"need_name"`
	Quantity  int       `json:"quantity" db:"quantity"`
	Unit      string    `json:"unit" db:"unit"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Resource 物资资源库存
type Resource struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Category   *string   `json:"category,omitempty" db:"category"`
	Icon       *string   `json:"icon,omitempty" db:"icon"`
	CurrentQty int       `json:"current_qty" db:"current_qty"`
	NeedQty    int       `json:"need_qty" db:"need_qty"`
	Unit       string    `json:"unit" db:"unit"`
	AreaCode   *string   `json:"area_code,omitempty" db:"area_code"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Shelter 安置点
type Shelter struct {
	ID           string        `json:"id" db:"id"`
	Name         string        `json:"name" db:"name"`
	Address      string        `json:"address" db:"address"`
	AreaCode     string        `json:"area_code" db:"area_code"`
	Lat          *float64      `json:"lat,omitempty" db:"lat"`
	Lng          *float64      `json:"lng,omitempty" db:"lng"`
	Capacity     int           `json:"capacity" db:"capacity"`
	Occupied     int           `json:"occupied" db:"occupied"`
	Status       ShelterStatus `json:"status" db:"status"`
	ContactPhone *string       `json:"contact_phone,omitempty" db:"contact_phone"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

// RescueTeam 救援队伍
type RescueTeam struct {
	ID              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	TeamType        TeamType  `json:"team_type" db:"team_type"`
	MembersCount    int       `json:"members_count" db:"members_count"`
	Status          TeamStatus `json:"status" db:"status"`
	AreaCode        *string   `json:"area_code,omitempty" db:"area_code"`
	TasksCount      int       `json:"tasks_count" db:"tasks_count"`
	CompletedCount  int       `json:"completed_count" db:"completed_count"`
	ContactPhone    *string   `json:"contact_phone,omitempty" db:"contact_phone"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Match P2P匹配记录
type Match struct {
	ID                      string     `json:"id" db:"id"`
	MatchNo                 string     `json:"match_no" db:"match_no"`
	RequestID               string     `json:"request_id" db:"request_id"`
	VolunteerID             string     `json:"volunteer_id" db:"volunteer_id"`
	Status                  MatchStatus `json:"status" db:"status"`
	RiskLevel               Urgency    `json:"risk_level" db:"risk_level"`
	DistanceKM              *float64   `json:"distance_km,omitempty" db:"distance_km"`
	MatchRadiusUsed         *float64   `json:"match_radius_used,omitempty" db:"match_radius_used"`
	ETAMin                  *int       `json:"eta_min,omitempty" db:"eta_min"`
	TimeoutSec              int        `json:"timeout_sec" db:"timeout_sec"`
	ElapsedSec              int        `json:"elapsed_sec" db:"elapsed_sec"`
	VirtualPhone            *string    `json:"virtual_phone,omitempty" db:"virtual_phone"`
	RelayHop                int        `json:"relay_hop" db:"relay_hop"`
	RelayGroupID            *string    `json:"relay_group_id,omitempty" db:"relay_group_id"`
	IsCooperative           bool       `json:"is_cooperative" db:"is_cooperative"`
	CooperativeGroupID      *string    `json:"cooperative_group_id,omitempty" db:"cooperative_group_id"`
	WaitingReason           *string    `json:"waiting_reason,omitempty" db:"waiting_reason"`
	WaitingETAMin           *int       `json:"waiting_eta_min,omitempty" db:"waiting_eta_min"`
	UnreachableDetectedAt   *time.Time `json:"unreachable_detected_at,omitempty" db:"unreachable_detected_at"`
	LocationSharingEnabled  bool       `json:"location_sharing_enabled" db:"location_sharing_enabled"`
	MatchTime               time.Time  `json:"match_time" db:"match_time"`
	AcceptTime              *time.Time `json:"accept_time,omitempty" db:"accept_time"`
	ArriveTime              *time.Time `json:"arrive_time,omitempty" db:"arrive_time"`
	CompleteTime            *time.Time `json:"complete_time,omitempty" db:"complete_time"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
}

// VirtualPhoneBinding AXB虚拟号码绑定
type VirtualPhoneBinding struct {
	ID                    string        `json:"id" db:"id"`
	MatchID               string        `json:"match_id" db:"match_id"`
	Provider              string        `json:"provider" db:"provider"`
	ProviderBindID        *string       `json:"provider_bind_id,omitempty" db:"provider_bind_id"`
	VirtualPhone          string        `json:"virtual_phone" db:"virtual_phone"`
	CallerPhoneEncrypted  []byte        `json:"-" db:"caller_phone_encrypted"`
	CalleePhoneEncrypted  []byte        `json:"-" db:"callee_phone_encrypted"`
	Status                BindingStatus `json:"status" db:"status"`
	BindTime              time.Time     `json:"bind_time" db:"bind_time"`
	ExpireTime            time.Time     `json:"expire_time" db:"expire_time"`
	UnbindTime            *time.Time    `json:"unbind_time,omitempty" db:"unbind_time"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
}

// CallRecord 通话记录
type CallRecord struct {
	ID                    string     `json:"id" db:"id"`
	MatchID               string     `json:"match_id" db:"match_id"`
	BindingID             *string    `json:"binding_id,omitempty" db:"binding_id"`
	CallType              CallType   `json:"call_type" db:"call_type"`
	CallerRole            CallerRole `json:"caller_role" db:"caller_role"`
	CallerPhoneEncrypted  []byte     `json:"-" db:"caller_phone_encrypted"`
	DurationSec           int        `json:"duration_sec" db:"duration_sec"`
	Status                CallStatus `json:"status" db:"status"`
	StartTime             time.Time  `json:"start_time" db:"start_time"`
	EndTime               *time.Time `json:"end_time,omitempty" db:"end_time"`
	RecordingURL          *string    `json:"recording_url,omitempty" db:"recording_url"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
}

// Escalation 升级告警
type Escalation struct {
	ID          string           `json:"id" db:"id"`
	MatchID     string           `json:"match_id" db:"match_id"`
	RequestID   string           `json:"request_id" db:"request_id"`
	Reason      string           `json:"reason" db:"reason"`
	Level       EscalationLevel  `json:"level" db:"level"`
	Action      EscalationAction `json:"action" db:"action"`
	Description *string          `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	ResolvedAt  *time.Time       `json:"resolved_at,omitempty" db:"resolved_at"`
}

// NotificationTemplate 通知模板
type NotificationTemplate struct {
	ID          string    `json:"id" db:"id"`
	TemplateID  string    `json:"template_id" db:"template_id"`
	Title       string    `json:"title" db:"title"`
	Description *string   `json:"description,omitempty" db:"description"`
	Type        NotifType `json:"type" db:"type"`
	Fields      []byte    `json:"fields,omitempty" db:"fields"` // JSONB
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Subscription 用户订阅授权
type Subscription struct {
	ID           string     `json:"id" db:"id"`
	UserID       string     `json:"user_id" db:"user_id"`
	TemplateID   string     `json:"template_id" db:"template_id"`
	Type         NotifType  `json:"type" db:"type"`
	Status       string     `json:"status" db:"status"`
	SubscribedAt time.Time  `json:"subscribed_at" db:"subscribed_at"`
	ExpiredAt    *time.Time `json:"expired_at,omitempty" db:"expired_at"`
}

// Notification 通知记录
type Notification struct {
	ID         string       `json:"id" db:"id"`
	UserID     string       `json:"user_id" db:"user_id"`
	TemplateID *string      `json:"template_id,omitempty" db:"template_id"`
	Title      string       `json:"title" db:"title"`
	Body       string       `json:"body" db:"body"`
	Urgency    NotifUrgency `json:"urgency" db:"urgency"`
	ActionType *string      `json:"action_type,omitempty" db:"action_type"`
	ActionData []byte       `json:"action_data,omitempty" db:"action_data"` // JSONB
	ReadStatus bool         `json:"read_status" db:"read_status"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	ReadAt     *time.Time   `json:"read_at,omitempty" db:"read_at"`
}

// Announcement 公告
type Announcement struct {
	ID          string        `json:"id" db:"id"`
	Title       string        `json:"title" db:"title"`
	Content     *string       `json:"content,omitempty" db:"content"`
	Level       AnnounceLevel `json:"level" db:"level"`
	Source      string        `json:"source" db:"source"`
	Status      string        `json:"status" db:"status"`
	PublishTime time.Time     `json:"publish_time" db:"publish_time"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

// AuditLog 审计日志
type AuditLog struct {
	ID           string     `json:"id" db:"id"`
	UserID       *string    `json:"user_id,omitempty" db:"user_id"`
	Action       string     `json:"action" db:"action"`
	ResourceType *string    `json:"resource_type,omitempty" db:"resource_type"`
	ResourceID   *string    `json:"resource_id,omitempty" db:"resource_id"`
	Detail       []byte     `json:"detail,omitempty" db:"detail"` // JSONB
	BeforeData   []byte     `json:"before_data,omitempty" db:"before_data"` // JSONB
	AfterData    []byte     `json:"after_data,omitempty" db:"after_data"` // JSONB
	IP           *string    `json:"ip,omitempty" db:"ip"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// RescueUpdate 救援动态
type RescueUpdate struct {
	ID               string    `json:"id" db:"id"`
	DisasterEventID  *string   `json:"disaster_event_id,omitempty" db:"disaster_event_id"`
	Title            string    `json:"title" db:"title"`
	Description      *string   `json:"description,omitempty" db:"description"`
	UpdateTime       time.Time `json:"update_time" db:"update_time"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// SOSTrigger SOS触发记录
type SOSTrigger struct {
	ID             string         `json:"id" db:"id"`
	UserID         *string        `json:"user_id,omitempty" db:"user_id"`
	GuestPhone     *string        `json:"guest_phone,omitempty" db:"guest_phone"`
	Lat            *float64       `json:"lat,omitempty" db:"lat"`
	Lng            *float64       `json:"lng,omitempty" db:"lng"`
	LocationSource LocationSource `json:"location_source" db:"location_source"`
	LastKnownLat   *float64       `json:"last_known_lat,omitempty" db:"last_known_lat"`
	LastKnownLng   *float64       `json:"last_known_lng,omitempty" db:"last_known_lng"`
	Address        *string        `json:"address,omitempty" db:"address"`
	Status         SOSStatus      `json:"status" db:"status"`
	TriggerTime    time.Time      `json:"trigger_time" db:"trigger_time"`
	ConfirmTime    *time.Time     `json:"confirm_time,omitempty" db:"confirm_time"`
	CancelTime     *time.Time     `json:"cancel_time,omitempty" db:"cancel_time"`
	FalseAlarm     bool           `json:"false_alarm" db:"false_alarm"`
	RequestID      *string        `json:"request_id,omitempty" db:"request_id"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// TaskCompletion 任务完成验证
type TaskCompletion struct {
	ID                  string           `json:"id" db:"id"`
	MatchID             string           `json:"match_id" db:"match_id"`
	RequestID           string           `json:"request_id" db:"request_id"`
	VolunteerID         string           `json:"volunteer_id" db:"volunteer_id"`
	Status              CompletionStatus `json:"status" db:"status"`
	SubmitTime          time.Time        `json:"submit_time" db:"submit_time"`
	SubmitPhotos        []string         `json:"submit_photos,omitempty" db:"submit_photos"`
	SubmitNote          *string          `json:"submit_note,omitempty" db:"submit_note"`
	ConfirmTime         *time.Time       `json:"confirm_time,omitempty" db:"confirm_time"`
	ConfirmMethod       *string          `json:"confirm_method,omitempty" db:"confirm_method"`
	DisputeNote         *string          `json:"dispute_note,omitempty" db:"dispute_note"`
	AutoConfirmDeadline *time.Time       `json:"auto_confirm_deadline,omitempty" db:"auto_confirm_deadline"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
}

// ResourceBatch 物资批次
type ResourceBatch struct {
	ID             string         `json:"id" db:"id"`
	ResourceID     *string        `json:"resource_id,omitempty" db:"resource_id"`
	BatchNo        string         `json:"batch_no" db:"batch_no"`
	Name           string         `json:"name" db:"name"`
	Category       *string        `json:"category,omitempty" db:"category"`
	Source         ResourceSource `json:"source" db:"source"`
	SourceOrg      *string        `json:"source_org,omitempty" db:"source_org"`
	Quantity       int            `json:"quantity" db:"quantity"`
	Unit           string         `json:"unit" db:"unit"`
	Status         BatchStatus    `json:"status" db:"status"`
	ProduceDate    *time.Time     `json:"produce_date,omitempty" db:"produce_date"`
	ExpireDate     *time.Time     `json:"expire_date,omitempty" db:"expire_date"`
	DepartTime     *time.Time     `json:"depart_time,omitempty" db:"depart_time"`
	ArriveTime     *time.Time     `json:"arrive_time,omitempty" db:"arrive_time"`
	DistributeTime *time.Time     `json:"distribute_time,omitempty" db:"distribute_time"`
	Destination    *string        `json:"destination,omitempty" db:"destination"`
	TransportNo    *string        `json:"transport_no,omitempty" db:"transport_no"`
	Remark         *string        `json:"remark,omitempty" db:"remark"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// Review 评价记录
type Review struct {
	ID            string          `json:"id" db:"id"`
	MatchID       string          `json:"match_id" db:"match_id"`
	Direction     ReviewDirection `json:"direction" db:"direction"`
	ReviewerID    *string         `json:"reviewer_id,omitempty" db:"reviewer_id"`
	RevieweeID    *string         `json:"reviewee_id,omitempty" db:"reviewee_id"`
	ResponseSpeed int             `json:"response_speed" db:"response_speed"`
	Attitude      int             `json:"attitude" db:"attitude"`
	Quality       int             `json:"quality" db:"quality"`
	OverallRating float64         `json:"overall_rating" db:"overall_rating"`
	TextContent   *string         `json:"text_content,omitempty" db:"text_content"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// DisasterAlert 灾情预警
type DisasterAlert struct {
	ID             string      `json:"id" db:"id"`
	AlertNo        string      `json:"alert_no" db:"alert_no"`
	Source         AlertSource `json:"source" db:"source"`
	SourceOrg      *string     `json:"source_org,omitempty" db:"source_org"`
	AlertType      string      `json:"alert_type" db:"alert_type"`
	Level          AlertLevel  `json:"level" db:"level"`
	Title          string      `json:"title" db:"title"`
	Content        string      `json:"content" db:"content"`
	AffectedAreas  []string    `json:"affected_areas,omitempty" db:"affected_areas"`
	PublishTime    time.Time   `json:"publish_time" db:"publish_time"`
	EffectiveUntil *time.Time  `json:"effective_until,omitempty" db:"effective_until"`
	Status         string      `json:"status" db:"status"`
	ExternalURL    *string     `json:"external_url,omitempty" db:"external_url"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
}

// AreaDensityConfig 区域密度配置
type AreaDensityConfig struct {
	ID            int         `json:"id" db:"id"`
	AreaCode      string      `json:"area_code" db:"area_code"`
	Density       AreaDensity `json:"density" db:"density"`
	MatchRadiusKM float64     `json:"match_radius_km" db:"match_radius_km"`
	RelayEnabled  bool        `json:"relay_enabled" db:"relay_enabled"`
	CreatedAt     time.Time   `json:"created_at" db:"created_at"`
}

// NotificationPreference 通知订阅偏好
type NotificationPreference struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	TemplateID string    `json:"template_id" db:"template_id"`
	Enabled    bool      `json:"enabled" db:"enabled"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// VolunteerDailyStats 志愿者每日统计
type VolunteerDailyStats struct {
	ID             int       `json:"id" db:"id"`
	VolunteerID    string    `json:"volunteer_id" db:"volunteer_id"`
	StatDate       time.Time `json:"stat_date" db:"stat_date"`
	AcceptedCount  int       `json:"accepted_count" db:"accepted_count"`
	CompletedCount int       `json:"completed_count" db:"completed_count"`
	AbandonedCount int       `json:"abandoned_count" db:"abandoned_count"`
	AbandonRate    float64   `json:"abandon_rate" db:"abandon_rate"`
}

// MatchRelay 接力匹配链路
type MatchRelay struct {
	ID              string     `json:"id" db:"id"`
	RelayGroupID    string     `json:"relay_group_id" db:"relay_group_id"`
	RequestID       string     `json:"request_id" db:"request_id"`
	HopOrder        int        `json:"hop_order" db:"hop_order"`
	FromVolunteerID *string    `json:"from_volunteer_id,omitempty" db:"from_volunteer_id"`
	ToVolunteerID   string     `json:"to_volunteer_id" db:"to_volunteer_id"`
	MatchID         *string    `json:"match_id,omitempty" db:"match_id"`
	DistanceKM      *float64   `json:"distance_km,omitempty" db:"distance_km"`
	Status          string     `json:"status" db:"status"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// MatchReinforcement 增援请求
type MatchReinforcement struct {
	ID                    string          `json:"id" db:"id"`
	MatchID               string          `json:"match_id" db:"match_id"`
	RequestID             string          `json:"request_id" db:"request_id"`
	RequesterVolunteerID  string          `json:"requester_volunteer_id" db:"requester_volunteer_id"`
	Reason                string          `json:"reason" db:"reason"`
	NeededCount           int             `json:"needed_count" db:"needed_count"`
	Status                ReinforceStatus `json:"status" db:"status"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	ResolvedAt            *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
}

// VolunteerLocationTrack 志愿者实时定位轨迹
type VolunteerLocationTrack struct {
	ID          string    `json:"id" db:"id"`
	MatchID     string    `json:"match_id" db:"match_id"`
	VolunteerID string    `json:"volunteer_id" db:"volunteer_id"`
	Lat         float64   `json:"lat" db:"lat"`
	Lng         float64   `json:"lng" db:"lng"`
	Accuracy    *float64  `json:"accuracy,omitempty" db:"accuracy"`
	Speed       *float64  `json:"speed,omitempty" db:"speed"`
	Heading     *float64  `json:"heading,omitempty" db:"heading"`
	RecordedAt  time.Time `json:"recorded_at" db:"recorded_at"`
}

// AutoTriggerRule 自动触发规则
type AutoTriggerRule struct {
	ID                  int           `json:"id" db:"id"`
	RuleName            string        `json:"rule_name" db:"rule_name"`
	ConditionAlertLevel AlertLevel    `json:"condition_alert_level" db:"condition_alert_level"`
	ConditionArea       *string       `json:"condition_area,omitempty" db:"condition_area"`
	ActionType          TriggerAction `json:"action_type" db:"action_type"`
	ActionParams        []byte        `json:"action_params,omitempty" db:"action_params"` // JSONB
	Enabled             bool          `json:"enabled" db:"enabled"`
	TriggeredCount      int           `json:"triggered_count" db:"triggered_count"`
	LastTriggeredAt     *time.Time    `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
	CreatedAt           time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at" db:"updated_at"`
}

// EvaluationReminder 评价提醒
type EvaluationReminder struct {
	ID           string    `json:"id" db:"id"`
	MatchID      string    `json:"match_id" db:"match_id"`
	UserID       string    `json:"user_id" db:"user_id"`
	ReminderType string    `json:"reminder_type" db:"reminder_type"`
	SentAt       time.Time `json:"sent_at" db:"sent_at"`
	Evaluated    bool      `json:"evaluated" db:"evaluated"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
