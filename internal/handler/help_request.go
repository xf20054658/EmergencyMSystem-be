package handler

import (
	"fmt"
	"time"

	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/internal/models"
	"emergency-msystem-backend/internal/service"
	"emergency-msystem-backend/internal/websocket"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// HelpRequestHandler 求助处理器
type HelpRequestHandler struct {
	matchEngine *service.MatchEngine
	wsHub       *websocket.Hub
}

// NewHelpRequestHandler 创建求助处理器
func NewHelpRequestHandler(matchEngine *service.MatchEngine, wsHub *websocket.Hub) *HelpRequestHandler {
	return &HelpRequestHandler{matchEngine: matchEngine, wsHub: wsHub}
}

// Create 创建求助
func (h *HelpRequestHandler) Create(c *gin.Context) {
	var req dto.CreateHelpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	id := uuid.New().String()
	requestNo := fmt.Sprintf("HR%s-%s", time.Now().Format("20060102"), uuid.New().String()[:4])

	// 路由决策
	route := "p2p"
	if req.Urgency == "critical" {
		route = "centralized"
	}

	proxyNotifyMethod := req.ProxyNotifyMethod
	if proxyNotifyMethod == "" {
		proxyNotifyMethod = "sms"
	}

	_, err := database.Pool.Exec(c, `
		INSERT INTO help_requests (id, request_no, user_id, disaster_event_id, area_code, address,
			lat, lng, content, urgency, people_count, status, route, water_level, images,
			is_proxy, proxy_phone, proxy_name, proxy_notify_method,
			vulnerable_groups, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'pending',$12,$13,$14,$15,$16,$17,$18,$19,NOW(),NOW())`,
		id, requestNo, userID, req.DisasterEventID, req.AreaCode, req.Address,
		req.Lat, req.Lng, req.Content, req.Urgency, req.PeopleCount,
		route, req.WaterLevel, req.Images,
		req.IsProxy, req.ProxyPhone, req.ProxyName, proxyNotifyMethod,
		req.VulnerableGroups,
	)
	if err != nil {
		response.InternalError(c, "create help request failed")
		return
	}

	// 创建物资需求
	for _, need := range req.Needs {
		unit := need.Unit
		if unit == "" {
			unit = "个"
		}
		_, _ = database.Pool.Exec(c, `
			INSERT INTO request_needs (request_id, need_name, quantity, unit, created_at)
			VALUES ($1,$2,$3,$4,NOW())
		`, id, need.NeedName, need.Quantity, unit)
	}

	// 非 critical 请求提交到匹配引擎
	if route == "p2p" {
		lat := 0.0
		lng := 0.0
		if req.Lat != nil {
			lat = *req.Lat
		}
		if req.Lng != nil {
			lng = *req.Lng
		}

		var vulnerableGroups []models.VulnerableGroup
		for _, vg := range req.VulnerableGroups {
			vulnerableGroups = append(vulnerableGroups, models.VulnerableGroup(vg))
		}

		h.matchEngine.Submit(id, models.Urgency(req.Urgency), req.AreaCode, lat, lng, vulnerableGroups)
	}

	// 通知指挥中心
	h.wsHub.SendToCommand(websocket.MsgTypeNewHelpRequest, map[string]interface{}{
		"id":         id,
		"request_no": requestNo,
		"urgency":    req.Urgency,
		"route":      route,
		"area_code":  req.AreaCode,
		"address":    req.Address,
	})

	response.Created(c, map[string]interface{}{
		"id":         id,
		"request_no": requestNo,
		"route":      route,
		"status":     "pending",
	})
}

// List 求助列表
func (h *HelpRequestHandler) List(c *gin.Context) {
	var query dto.HelpRequestListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query params")
		return
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	// 构建查询
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if query.Status != "" {
		where += fmt.Sprintf(" AND r.status = $%d", argIdx)
		args = append(args, query.Status)
		argIdx++
	}
	if query.Urgency != "" {
		where += fmt.Sprintf(" AND r.urgency = $%d", argIdx)
		args = append(args, query.Urgency)
		argIdx++
	}
	if query.AreaCode != "" {
		where += fmt.Sprintf(" AND r.area_code = $%d", argIdx)
		args = append(args, query.AreaCode)
		argIdx++
	}
	if query.UserID != "" {
		where += fmt.Sprintf(" AND r.user_id = $%d", argIdx)
		args = append(args, query.UserID)
		argIdx++
	}
	if query.Search != "" {
		where += fmt.Sprintf(" AND (r.content ILIKE $%d OR r.address ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+query.Search+"%")
		argIdx++
	}

	// 总数
	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM help_requests r %s", where)
	database.Pool.QueryRow(c, countSQL, args...).Scan(&total)

	// 数据
	offset := (query.Page - 1) * query.PageSize
	dataSQL := fmt.Sprintf(`
		SELECT r.id, r.request_no, r.user_id, COALESCE(u.nickname, ''), COALESCE(u.phone_masked, ''),
			r.area_code, COALESCE(a.name, ''), r.address, COALESCE(r.lat, 0), COALESCE(r.lng, 0),
			r.content, r.urgency, r.people_count, r.status, r.route,
			COALESCE(r.water_level::text, ''), r.images,
			r.is_proxy, COALESCE(r.proxy_phone, ''), COALESCE(r.proxy_name, ''),
			r.proxy_notify_method, r.vulnerable_groups,
			r.created_at, r.updated_at
		FROM help_requests r
		LEFT JOIN users u ON r.user_id = u.id
		LEFT JOIN areas a ON r.area_code = a.code
		%s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, query.PageSize, offset)

	rows, err := database.Pool.Query(c, dataSQL, args...)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	var items []dto.HelpRequestResponse
	for rows.Next() {
		var item dto.HelpRequestResponse
		var lat, lng float64
		var images []string
		var vulnerableGroups []string
		var createdAt, updatedAt time.Time
		var resolvedAt *time.Time
		var waterLevelStr string

		err := rows.Scan(
			&item.ID, &item.RequestNo, &item.UserID, &item.Nickname, &item.PhoneMasked,
			&item.AreaCode, &item.AreaName, &item.Address, &lat, &lng,
			&item.Content, &item.Urgency, &item.PeopleCount, &item.Status, &item.Route,
			&waterLevelStr, &images,
			&item.IsProxy, &item.ProxyPhone, &item.ProxyName,
			&item.ProxyNotifyMethod, &vulnerableGroups,
			&createdAt, &updatedAt,
		)
		if err != nil {
			continue
		}

		item.Lat = lat
		item.Lng = lng
		item.Images = images
		item.VulnerableGroups = vulnerableGroups
		if waterLevelStr != "" && waterLevelStr != "none" {
			item.WaterLevel = waterLevelStr
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		if resolvedAt != nil {
			item.ResolvedAt = resolvedAt.Format(time.RFC3339)
		}

		items = append(items, item)
	}

	response.Paginated(c, items, total, query.Page, query.PageSize)
}

// Get 获取求助详情
func (h *HelpRequestHandler) Get(c *gin.Context) {
	id := c.Param("id")

	// 校验 ID 是否为有效 UUID，避免数据库报 500
	if _, parseErr := uuid.Parse(id); parseErr != nil {
		response.NotFound(c, "help request not found")
		return
	}

	var item dto.HelpRequestResponse
	var lat, lng float64
	var images []string
	var vulnerableGroups []string
	var createdAt, updatedAt time.Time
	var waterLevelStr string

	err := database.Pool.QueryRow(c, `
		SELECT r.id, r.request_no, r.user_id, COALESCE(u.nickname, ''), COALESCE(u.phone_masked, ''),
			r.area_code, COALESCE(a.name, ''), r.address, COALESCE(r.lat, 0), COALESCE(r.lng, 0),
			r.content, r.urgency, r.people_count, r.status, r.route,
			COALESCE(r.water_level::text, ''), r.images,
			r.is_proxy, COALESCE(r.proxy_phone, ''), COALESCE(r.proxy_name, ''),
			r.proxy_notify_method, r.vulnerable_groups,
			r.created_at, r.updated_at
		FROM help_requests r
		LEFT JOIN users u ON r.user_id = u.id
		LEFT JOIN areas a ON r.area_code = a.code
		WHERE r.id = $1
	`, id).Scan(
		&item.ID, &item.RequestNo, &item.UserID, &item.Nickname, &item.PhoneMasked,
		&item.AreaCode, &item.AreaName, &item.Address, &lat, &lng,
		&item.Content, &item.Urgency, &item.PeopleCount, &item.Status, &item.Route,
		&waterLevelStr, &images,
		&item.IsProxy, &item.ProxyPhone, &item.ProxyName,
		&item.ProxyNotifyMethod, &vulnerableGroups,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "help request not found")
			return
		}
		response.InternalError(c, "query failed")
		return
	}

	item.Lat = lat
	item.Lng = lng
	item.Images = images
	item.VulnerableGroups = vulnerableGroups
	if waterLevelStr != "" && waterLevelStr != "none" {
		item.WaterLevel = waterLevelStr
	}
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)

	// 获取物资需求
	needRows, err := database.Pool.Query(c, `
		SELECT need_name, quantity, unit FROM request_needs WHERE request_id = $1
	`, id)
	if err == nil {
		defer needRows.Close()
		for needRows.Next() {
			var need dto.RequestNeedItem
			needRows.Scan(&need.NeedName, &need.Quantity, &need.Unit)
			item.Needs = append(item.Needs, need)
		}
	}

	// 获取关联匹配
	matchRows, err := database.Pool.Query(c, `
		SELECT m.id, m.match_no, m.status, v.display_name, v.tier::text, m.distance_km
		FROM matches m
		LEFT JOIN volunteers v ON m.volunteer_id = v.id
		WHERE m.request_id = $1
		ORDER BY m.match_time DESC
	`, id)
	if err == nil {
		defer matchRows.Close()
		for matchRows.Next() {
			var m dto.MatchBrief
			var dist float64
			matchRows.Scan(&m.ID, &m.MatchNo, &m.Status, &m.VolunteerName, &m.VolunteerTier, &dist)
			m.DistanceKM = dist
			item.Matches = append(item.Matches, m)
		}
	}

	response.Success(c, item)
}

// Update 更新求助
func (h *HelpRequestHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateHelpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	setClauses := "updated_at = NOW()"
	args := []interface{}{id}
	argIdx := 2

	if req.Address != nil {
		setClauses += fmt.Sprintf(", address = $%d", argIdx)
		args = append(args, *req.Address)
		argIdx++
	}
	if req.Content != nil {
		setClauses += fmt.Sprintf(", content = $%d", argIdx)
		args = append(args, *req.Content)
		argIdx++
	}
	if req.Status != nil {
		setClauses += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, *req.Status)
		argIdx++

		if *req.Status == "resolved" {
			setClauses += ", resolved_at = NOW()"
		}
	}

	_, err := database.Pool.Exec(c, fmt.Sprintf(
		"UPDATE help_requests SET %s WHERE id = $1", setClauses,
	), args...)
	if err != nil {
		response.InternalError(c, "update failed")
		return
	}

	response.Success(c, nil)
}

// Cancel 取消求助
func (h *HelpRequestHandler) Cancel(c *gin.Context) {
	id := c.Param("id")

	_, err := database.Pool.Exec(c, `
		UPDATE help_requests SET status = 'cancelled', updated_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		response.InternalError(c, "cancel failed")
		return
	}

	response.Success(c, nil)
}
