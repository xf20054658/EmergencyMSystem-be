package handler

import (
	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// DashboardHandler 仪表盘处理器
type DashboardHandler struct{}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// GetKPI 获取仪表盘 KPI
func (h *DashboardHandler) GetKPI(c *gin.Context) {
	var kpi struct {
		TotalHelpRequests   int64 `json:"total_help_requests"`
		PendingHelpRequests int64 `json:"pending_help_requests"`
		ProcessingRequests  int64 `json:"processing_requests"`
		ActiveMatches       int64 `json:"active_matches"`
		TotalVolunteers     int64 `json:"total_volunteers"`
		AvailableVolunteers int64 `json:"available_volunteers"`
		TotalShelters       int64 `json:"total_shelters"`
		AvailableShelters   int64 `json:"available_shelters"`
		TotalRescueTeams    int64 `json:"total_rescue_teams"`
		ActiveAlerts        int64 `json:"active_alerts"`
	}

	// 并发查询各指标
	ch := make(chan func(), 10)

	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests`).Scan(&kpi.TotalHelpRequests)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests WHERE status = 'pending'`).Scan(&kpi.PendingHelpRequests)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests WHERE status = 'processing'`).Scan(&kpi.ProcessingRequests)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM matches WHERE status IN ('accepted','enroute','waiting')`).Scan(&kpi.ActiveMatches)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM volunteers`).Scan(&kpi.TotalVolunteers)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM volunteers WHERE status = 'available'`).Scan(&kpi.AvailableVolunteers)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM shelters`).Scan(&kpi.TotalShelters)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM shelters WHERE status = 'open'`).Scan(&kpi.AvailableShelters)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM rescue_teams`).Scan(&kpi.TotalRescueTeams)
		ch <- nil
	}()
	go func() {
		database.Pool.QueryRow(c, `SELECT COUNT(*) FROM disaster_alerts WHERE status = 'active'`).Scan(&kpi.ActiveAlerts)
		ch <- nil
	}()

	// 等待所有查询完成
	for i := 0; i < 10; i++ {
		<-ch
	}

	response.Success(c, kpi)
}

// GetUrgencyDistribution 紧急程度分布
func (h *DashboardHandler) GetUrgencyDistribution(c *gin.Context) {
	var dist struct {
		Critical int64 `json:"critical"`
		High     int64 `json:"high"`
		Medium   int64 `json:"medium"`
		Low      int64 `json:"low"`
	}

	database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests WHERE urgency = 'critical'`).Scan(&dist.Critical)
	database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests WHERE urgency = 'high'`).Scan(&dist.High)
	database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests WHERE urgency = 'medium'`).Scan(&dist.Medium)
	database.Pool.QueryRow(c, `SELECT COUNT(*) FROM help_requests WHERE urgency = 'low'`).Scan(&dist.Low)

	response.Success(c, dist)
}

// GetMatchStats 匹配效率统计
func (h *DashboardHandler) GetMatchStats(c *gin.Context) {
	var stats struct {
		TotalMatches     int64   `json:"total_matches"`
		CompletedCount    int64   `json:"completed_count"`
		EscalatedCount    int64   `json:"escalated_count"`
		P2PCoverageRate   float64 `json:"p2p_coverage_rate"`
		EscalationRate    float64 `json:"escalation_rate"`
	}

	// 使用预定义视图
	err := database.Pool.QueryRow(c, `
		SELECT total_matches, completed_count, escalated_count,
			p2p_coverage_rate, escalation_rate
		FROM v_match_stats
	`).Scan(&stats.TotalMatches, &stats.CompletedCount, &stats.EscalatedCount,
		&stats.P2PCoverageRate, &stats.EscalationRate)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}

	response.Success(c, stats)
}

// GetShelters 安置点列表
func (h *DashboardHandler) GetShelters(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT id, name, address, area_code, capacity, occupied, available,
			occupancy_rate, status::text, capacity_level
		FROM v_shelters_capacity
		ORDER BY occupancy_rate DESC
	`)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	var shelters []map[string]interface{}
	for rows.Next() {
		var id, name, address, areaCode, status, capLevel string
		var capacity, occupied, available int
		var rate float64

		rows.Scan(&id, &name, &address, &areaCode, &capacity, &occupied, &available, &rate, &status, &capLevel)

		shelters = append(shelters, map[string]interface{}{
			"id":             id,
			"name":           name,
			"address":        address,
			"area_code":      areaCode,
			"capacity":       capacity,
			"occupied":       occupied,
			"available":      available,
			"occupancy_rate": rate,
			"status":         status,
			"capacity_level": capLevel,
		})
	}

	response.Success(c, shelters)
}

// GetResources 物资列表
func (h *DashboardHandler) GetResources(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT id, name, COALESCE(category, ''), COALESCE(icon, ''), current_qty,
			need_qty, unit, COALESCE(area_code, '')
		FROM resources
		ORDER BY current_qty DESC
	`)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	type ResourceItem struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Category   string `json:"category"`
		Icon       string `json:"icon"`
		CurrentQty int    `json:"current_qty"`
		NeedQty    int    `json:"need_qty"`
		Unit       string `json:"unit"`
		AreaCode   string `json:"area_code"`
	}

	var items []ResourceItem
	for rows.Next() {
		var item ResourceItem
		rows.Scan(&item.ID, &item.Name, &item.Category, &item.Icon,
			&item.CurrentQty, &item.NeedQty, &item.Unit, &item.AreaCode)
		items = append(items, item)
	}

	response.Success(c, items)
}

// GetRescueTeams 救援队伍列表
func (h *DashboardHandler) GetRescueTeams(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT id, name, team_type::text, members_count, status::text,
			COALESCE(area_code, ''), tasks_count, completed_count
		FROM rescue_teams
		ORDER BY members_count DESC
	`)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	type TeamItem struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		TeamType       string `json:"team_type"`
		MembersCount   int    `json:"members_count"`
		Status         string `json:"status"`
		AreaCode       string `json:"area_code"`
		TasksCount     int    `json:"tasks_count"`
		CompletedCount int    `json:"completed_count"`
	}

	var items []TeamItem
	for rows.Next() {
		var item TeamItem
		rows.Scan(&item.ID, &item.Name, &item.TeamType, &item.MembersCount,
			&item.Status, &item.AreaCode, &item.TasksCount, &item.CompletedCount)
		items = append(items, item)
	}

	response.Success(c, items)
}

// GetEscalations 升级告警列表
func (h *DashboardHandler) GetEscalations(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT e.id, e.reason, e.level::text, e.action::text, e.created_at
		FROM escalations e
		WHERE e.resolved_at IS NULL
		ORDER BY e.created_at DESC
		LIMIT 50
	`)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	type EscalationItem struct {
		ID        string `json:"id"`
		Reason    string `json:"reason"`
		Level     string `json:"level"`
		Action    string `json:"action"`
		CreatedAt string `json:"created_at"`
	}

	var items []EscalationItem
	for rows.Next() {
		var item EscalationItem
		var createdAt interface{}
		rows.Scan(&item.ID, &item.Reason, &item.Level, &item.Action, &createdAt)
		if t, ok := createdAt.(interface{ Format(string) string }); ok {
			item.CreatedAt = t.Format("2006-01-02T15:04:05Z")
		}
		items = append(items, item)
	}

	response.Success(c, items)
}
