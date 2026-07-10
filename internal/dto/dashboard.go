package dto

// --- Dashboard DTOs ---

// DashboardKPI 仪表盘KPI
type DashboardKPI struct {
	TotalHelpRequests   int64 `json:"total_help_requests"`
	PendingHelpRequests int64 `json:"pending_help_requests"`
	ActiveMatches       int64 `json:"active_matches"`
	TotalVolunteers     int64 `json:"total_volunteers"`
	ActiveVolunteers    int64 `json:"active_volunteers"`
	AvailableShelters   int64 `json:"available_shelters"`
	TotalShelterBeds    int64 `json:"total_shelter_beds"`
	OccupiedBeds        int64 `json:"occupied_beds"`
	UnreadAlerts        int64 `json:"unread_alerts"`
}

// UrgencyDistribution 紧急程度分布
type UrgencyDistribution struct {
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
}

// MatchStats 匹配效率统计
type MatchStats struct {
	TotalMatches      int64   `json:"total_matches"`
	CompletedCount    int64   `json:"completed_count"`
	EscalatedCount    int64   `json:"escalated_count"`
	TimeoutCount      int64   `json:"timeout_count"`
	UnreachableCount  int64   `json:"unreachable_count"`
	WaitingCount      int64   `json:"waiting_count"`
	CooperativeCount  int64   `json:"cooperative_count"`
	RelayCount        int64   `json:"relay_count"`
	AvgAcceptSec      float64 `json:"avg_accept_sec"`
	AvgDistanceKM     float64 `json:"avg_distance_km"`
	P2PCoverageRate   float64 `json:"p2p_coverage_rate"`
	EscalationRate    float64 `json:"escalation_rate"`
	UnreachableRate   float64 `json:"unreachable_rate"`
}
