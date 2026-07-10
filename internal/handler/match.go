package handler

import (
	"fmt"
	"time"

	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/internal/service"
	"emergency-msystem-backend/internal/websocket"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MatchHandler 匹配处理器
type MatchHandler struct {
	matchEngine *service.MatchEngine
	wsHub       *websocket.Hub
}

// NewMatchHandler 创建匹配处理器
func NewMatchHandler(matchEngine *service.MatchEngine, wsHub *websocket.Hub) *MatchHandler {
	return &MatchHandler{matchEngine: matchEngine, wsHub: wsHub}
}

// List 匹配列表
func (h *MatchHandler) List(c *gin.Context) {
	var query dto.MatchListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid params")
		return
	}
	if query.Page < 1 { query.Page = 1 }
	if query.PageSize < 1 || query.PageSize > 100 { query.PageSize = 20 }

	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if query.Status != "" {
		where += fmt.Sprintf(" AND m.status = $%d", argIdx)
		args = append(args, query.Status); argIdx++
	}
	if query.RequestID != "" {
		where += fmt.Sprintf(" AND m.request_id = $%d", argIdx)
		args = append(args, query.RequestID); argIdx++
	}
	if query.VolunteerID != "" {
		where += fmt.Sprintf(" AND m.volunteer_id = $%d", argIdx)
		args = append(args, query.VolunteerID); argIdx++
	}

	var total int64
	database.Pool.QueryRow(c, fmt.Sprintf("SELECT COUNT(*) FROM matches m %s", where), args...).Scan(&total)

	offset := (query.Page - 1) * query.PageSize
	dataSQL := fmt.Sprintf(`
		SELECT m.id, m.match_no, m.request_id, r.request_no, r.address, r.urgency::text,
			m.volunteer_id, v.display_name, v.tier::text, v.trust_score,
			m.status, m.risk_level::text, COALESCE(m.distance_km, 0),
			COALESCE(m.match_radius_used, 0), COALESCE(m.eta_min, 0),
			m.timeout_sec, m.elapsed_sec, COALESCE(m.virtual_phone, ''),
			m.relay_hop, m.is_cooperative,
			COALESCE(m.waiting_reason, ''), COALESCE(m.waiting_eta_min, 0),
			m.location_sharing_enabled,
			m.match_time, m.accept_time, m.arrive_time, m.complete_time,
			m.created_at
		FROM matches m
		LEFT JOIN help_requests r ON m.request_id = r.id
		LEFT JOIN volunteers v ON m.volunteer_id = v.id
		%s
		ORDER BY m.match_time DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, query.PageSize, offset)

	rows, err := database.Pool.Query(c, dataSQL, args...)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	var items []dto.MatchResponse
	for rows.Next() {
		var m dto.MatchResponse
		var dist, radius float64
		var matchTime time.Time
		var acceptTime, arriveTime, completeTime, unreachableDetected *time.Time
		var relayGroupID, coopGroupID *string

		rows.Scan(&m.ID, &m.MatchNo, &m.RequestID, &m.RequestNo, &m.RequestAddress, &m.RequestUrgency,
			&m.VolunteerID, &m.VolunteerName, &m.VolunteerTier, &m.VolunteerTrustScore,
			&m.Status, &m.RiskLevel, &dist, &radius, &m.ETAMin,
			&m.TimeoutSec, &m.ElapsedSec, &m.VirtualPhone,
			&m.RelayHop, &m.IsCooperative,
			&m.WaitingReason, &m.WaitingETAMin,
			&m.LocationSharingEnabled,
			&matchTime, &acceptTime, &arriveTime, &completeTime,
			&m.CreatedAt,
		)
		m.DistanceKM = dist
		m.MatchRadiusUsed = radius
		m.MatchTime = matchTime.Format(time.RFC3339)
		m.CreatedAt = matchTime.Format(time.RFC3339)
		if acceptTime != nil { m.AcceptTime = acceptTime.Format(time.RFC3339) }
		if arriveTime != nil { m.ArriveTime = arriveTime.Format(time.RFC3339) }
		if completeTime != nil { m.CompleteTime = completeTime.Format(time.RFC3339) }
		if relayGroupID != nil { m.RelayGroupID = *relayGroupID }
		if coopGroupID != nil { m.CooperativeGroupID = *coopGroupID }
		if unreachableDetected != nil { m.UnreachableDetectedAt = unreachableDetected.Format(time.RFC3339) }

		items = append(items, m)
	}

	response.Paginated(c, items, total, query.Page, query.PageSize)
}

// Get 获取匹配详情
func (h *MatchHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var m dto.MatchResponse
	var dist, radius float64
	var matchTime, createdAt time.Time
	var acceptTime, arriveTime, completeTime, unreachableDetected *time.Time
	var relayGroupID, coopGroupID, virtualPhone, waitingReason *string
	var etaMin, waitingETAMin *int

	err := database.Pool.QueryRow(c, `
		SELECT m.id, m.match_no, m.request_id, r.request_no, r.address, r.urgency::text,
			m.volunteer_id, v.display_name, v.tier::text, v.trust_score,
			m.status, m.risk_level::text, m.distance_km, m.match_radius_used, m.eta_min,
			m.timeout_sec, m.elapsed_sec, m.virtual_phone,
			m.relay_hop, m.is_cooperative, m.cooperative_group_id, m.relay_group_id,
			m.waiting_reason, m.waiting_eta_min, m.unreachable_detected_at,
			m.location_sharing_enabled,
			m.match_time, m.accept_time, m.arrive_time, m.complete_time, m.created_at
		FROM matches m
		LEFT JOIN help_requests r ON m.request_id = r.id
		LEFT JOIN volunteers v ON m.volunteer_id = v.id
		WHERE m.id = $1
	`, id).Scan(
		&m.ID, &m.MatchNo, &m.RequestID, &m.RequestNo, &m.RequestAddress, &m.RequestUrgency,
		&m.VolunteerID, &m.VolunteerName, &m.VolunteerTier, &m.VolunteerTrustScore,
		&m.Status, &m.RiskLevel, &dist, &radius, &etaMin,
		&m.TimeoutSec, &m.ElapsedSec, &virtualPhone,
		&m.RelayHop, &m.IsCooperative, &coopGroupID, &relayGroupID,
		&waitingReason, &waitingETAMin, &unreachableDetected,
		&m.LocationSharingEnabled,
		&matchTime, &acceptTime, &arriveTime, &completeTime, &createdAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "match not found")
			return
		}
		response.InternalError(c, "query failed")
		return
	}

	m.DistanceKM = dist
	m.MatchRadiusUsed = radius
	if etaMin != nil { m.ETAMin = *etaMin }
	if virtualPhone != nil { m.VirtualPhone = *virtualPhone }
	if relayGroupID != nil { m.RelayGroupID = *relayGroupID }
	if coopGroupID != nil { m.CooperativeGroupID = *coopGroupID }
	if waitingReason != nil { m.WaitingReason = *waitingReason }
	if waitingETAMin != nil { m.WaitingETAMin = *waitingETAMin }
	m.MatchTime = matchTime.Format(time.RFC3339)
	m.CreatedAt = createdAt.Format(time.RFC3339)
	if acceptTime != nil { m.AcceptTime = acceptTime.Format(time.RFC3339) }
	if arriveTime != nil { m.ArriveTime = arriveTime.Format(time.RFC3339) }
	if completeTime != nil { m.CompleteTime = completeTime.Format(time.RFC3339) }
	if unreachableDetected != nil { m.UnreachableDetectedAt = unreachableDetected.Format(time.RFC3339) }

	response.Success(c, m)
}

// Accept 志愿者接受匹配
func (h *MatchHandler) Accept(c *gin.Context) {
	id := c.Param("id")
	volunteerID, _ := c.Get("volunteer_id")

	var req dto.AcceptMatchRequest
	c.ShouldBindJSON(&req) // 可选

	now := time.Now()
	_, err := database.Pool.Exec(c, `
		UPDATE matches SET status = 'accepted', accept_time = $1, updated_at = $1,
			eta_min = COALESCE($2, eta_min)
		WHERE id = $3 AND volunteer_id = $4 AND status = 'pending'
	`, now, req.ETAMin, id, volunteerID)
	if err != nil {
		response.InternalError(c, "accept failed")
		return
	}

	// 更新志愿者统计
	_, _ = database.Pool.Exec(c, `
		INSERT INTO volunteer_daily_stats (volunteer_id, stat_date, accepted_count)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (volunteer_id, stat_date)
		DO UPDATE SET accepted_count = volunteer_daily_stats.accepted_count + 1
	`, volunteerID)

	// 通知求助者和指挥中心
	h.wsHub.SendToCommand(websocket.MsgTypeMatchUpdate, map[string]interface{}{
		"match_id": id,
		"status":   "accepted",
	})

	response.Success(c, nil)
}

// Reject 志愿者拒绝匹配
func (h *MatchHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	volunteerID, _ := c.Get("volunteer_id")

	_, err := database.Pool.Exec(c, `
		UPDATE matches SET status = 'rejected', updated_at = NOW()
		WHERE id = $1 AND volunteer_id = $2 AND status = 'pending'
	`, id, volunteerID)
	if err != nil {
		response.InternalError(c, "reject failed")
		return
	}

	response.Success(c, nil)
}

// SetEnroute 设为前往中
func (h *MatchHandler) SetEnroute(c *gin.Context) {
	id := c.Param("id")
	volunteerID, _ := c.Get("volunteer_id")

	_, err := database.Pool.Exec(c, `
		UPDATE matches SET status = 'enroute', arrive_time = NOW(), updated_at = NOW()
		WHERE id = $1 AND volunteer_id = $2 AND status IN ('accepted', 'waiting')
	`, id, volunteerID)
	if err != nil {
		response.InternalError(c, "set enroute failed")
		return
	}

	response.Success(c, nil)
}

// SetWaiting 设为等待中（道路受阻/交通堵塞）
func (h *MatchHandler) SetWaiting(c *gin.Context) {
	id := c.Param("id")
	volunteerID, _ := c.Get("volunteer_id")

	var req dto.SetWaitingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE matches SET status = 'waiting', waiting_reason = $1, waiting_eta_min = $2,
			updated_at = NOW()
		WHERE id = $3 AND volunteer_id = $4 AND status IN ('accepted', 'enroute')
	`, req.Reason, req.ETAMin, id, volunteerID)
	if err != nil {
		response.InternalError(c, "set waiting failed")
		return
	}

	h.wsHub.SendToCommand(websocket.MsgTypeMatchUpdate, map[string]interface{}{
		"match_id":    id,
		"status":      "waiting",
		"reason":      req.Reason,
		"eta_min":     req.ETAMin,
	})

	response.Success(c, nil)
}

// Complete 任务完成
func (h *MatchHandler) Complete(c *gin.Context) {
	id := c.Param("id")
	volunteerID, _ := c.Get("volunteer_id")

	var req dto.CompleteMatchRequest
	c.ShouldBindJSON(&req)

	var requestID string
	err := database.Pool.QueryRow(c,
		`SELECT request_id FROM matches WHERE id = $1 AND volunteer_id = $2`,
		id, volunteerID,
	).Scan(&requestID)
	if err != nil {
		response.NotFound(c, "match not found")
		return
	}

	now := time.Now()

	// 创建任务完成记录
	completionID := uuid.New().String()
	_, err = database.Pool.Exec(c, `
		INSERT INTO task_completions (id, match_id, request_id, volunteer_id, status,
			submit_time, submit_photos, submit_note, auto_confirm_deadline, created_at)
		VALUES ($1,$2,$3,$4,'submitted',$5,$6,$7,$8,NOW())
	`, completionID, id, requestID, volunteerID, now, req.SubmitPhotos, req.SubmitNote,
		now.Add(time.Duration(h.matchEngine.GetAutoConfirmHours())*time.Hour))
	if err != nil {
		response.InternalError(c, "create completion failed")
		return
	}

	// 更新匹配状态
	_, _ = database.Pool.Exec(c, `
		UPDATE matches SET complete_time = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)

	// 更新志愿者统计
	_, _ = database.Pool.Exec(c, `
		UPDATE volunteers SET completed_tasks = completed_tasks + 1 WHERE id = $1
	`, volunteerID)

	_, _ = database.Pool.Exec(c, `
		INSERT INTO volunteer_daily_stats (volunteer_id, stat_date, completed_count)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (volunteer_id, stat_date)
		DO UPDATE SET completed_count = volunteer_daily_stats.completed_count + 1
	`, volunteerID)

	response.Success(c, map[string]interface{}{
		"completion_id": completionID,
		"status":        "submitted",
	})
}

// ConfirmCompletion 确认任务完成
func (h *MatchHandler) ConfirmCompletion(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	var req dto.ConfirmCompletionRequest
	c.ShouldBindJSON(&req)

	if req.DisputeNote != "" {
		// 有异议
		_, err := database.Pool.Exec(c, `
			UPDATE task_completions SET status = 'disputed', dispute_note = $1
			WHERE match_id = $2 AND status = 'submitted'
		`, req.DisputeNote, id)
		if err != nil {
			response.InternalError(c, "dispute failed")
			return
		}
		response.Success(c, map[string]string{"status": "disputed"})
		return
	}

	// 确认
	_, err := database.Pool.Exec(c, `
		UPDATE task_completions SET status = 'confirmed', confirm_method = 'requester',
			confirm_time = NOW()
		WHERE match_id = $1 AND status = 'submitted'
		AND request_id IN (SELECT id FROM help_requests WHERE user_id = $2)
	`, id, userID)
	if err != nil {
		response.InternalError(c, "confirm failed")
		return
	}

	// 更新匹配和求助状态
	_, _ = database.Pool.Exec(c, `
		UPDATE matches SET status = 'completed', updated_at = NOW() WHERE id = $1
	`, id)

	_, _ = database.Pool.Exec(c, `
		UPDATE help_requests SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
		WHERE id = (SELECT request_id FROM matches WHERE id = $1)
	`, id)

	response.Success(c, map[string]string{"status": "confirmed"})
}

// Reinforce 增援请求
func (h *MatchHandler) Reinforce(c *gin.Context) {
	id := c.Param("id")
	volunteerID, _ := c.Get("volunteer_id")

	var req dto.ReinforceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	var requestID string
	database.Pool.QueryRow(c, `SELECT request_id FROM matches WHERE id = $1`, id).Scan(&requestID)

	rID := uuid.New().String()
	_, err := database.Pool.Exec(c, `
		INSERT INTO match_reinforcements (id, match_id, request_id, requester_volunteer_id,
			reason, needed_count, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,'pending',NOW())
	`, rID, id, requestID, volunteerID, req.Reason, req.NeededCount)
	if err != nil {
		response.InternalError(c, "reinforce failed")
		return
	}

	h.wsHub.SendToCommand(websocket.MsgTypeMatchUpdate, map[string]interface{}{
		"type":        "reinforce_request",
		"match_id":    id,
		"request_id":  requestID,
		"reason":      req.Reason,
		"needed_count": req.NeededCount,
	})

	response.Created(c, map[string]string{"id": rID})
}

// GPSReport 志愿者GPS上报
func (h *MatchHandler) GPSReport(c *gin.Context) {
	volunteerID, _ := c.Get("volunteer_id")

	var req dto.GPSReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	trackID := uuid.New().String()
	_, err := database.Pool.Exec(c, `
		INSERT INTO volunteer_location_tracks (id, match_id, volunteer_id, lat, lng,
			accuracy, speed, heading, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`, trackID, req.MatchID, volunteerID, req.Lat, req.Lng,
		req.Accuracy, req.Speed, req.Heading)
	if err != nil {
		response.InternalError(c, "gps report failed")
		return
	}

	// 推送GPS坐标给指挥中心和求助者
	h.wsHub.SendToCommand(websocket.MsgTypeGPSUpdate, map[string]interface{}{
		"match_id":    req.MatchID,
		"volunteer_id": volunteerID,
		"lat":         req.Lat,
		"lng":         req.Lng,
	})

	response.Success(c, nil)
}
