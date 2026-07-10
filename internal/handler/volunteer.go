package handler

import (
	"fmt"
	"time"

	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/internal/websocket"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// VolunteerHandler 志愿者处理器
type VolunteerHandler struct {
	wsHub *websocket.Hub
}

// NewVolunteerHandler 创建志愿者处理器
func NewVolunteerHandler(wsHub *websocket.Hub) *VolunteerHandler {
	return &VolunteerHandler{wsHub: wsHub}
}

// Register 注册成为志愿者
func (h *VolunteerHandler) Register(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req dto.CreateVolunteerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	// 检查是否已注册
	var existingID string
	err := database.Pool.QueryRow(c,
		`SELECT id FROM volunteers WHERE user_id = $1 LIMIT 1`, userID,
	).Scan(&existingID)
	if err == nil {
		response.Conflict(c, "already registered as volunteer")
		return
	}

	id := uuid.New().String()
	_, err = database.Pool.Exec(c, `
		INSERT INTO volunteers (id, user_id, display_name, tier, trust_score, status,
			bio, cert_no, area_code, lat, lng, created_at, updated_at)
		VALUES ($1,$2,$3,'tier3',60,'available',$4,$5,$6,$7,$8,NOW(),NOW())
	`, id, userID, req.DisplayName, req.Bio, req.CertNo, req.AreaCode, req.Lat, req.Lng)
	if err != nil {
		response.InternalError(c, "register failed")
		return
	}

	// 添加技能
	for _, skill := range req.Skills {
		prof := skill.Proficiency
		if prof == "" { prof = "basic" }
		database.Pool.Exec(c, `
			INSERT INTO volunteer_skills (volunteer_id, skill_name, proficiency, created_at)
			VALUES ($1,$2,$3,NOW())
		`, id, skill.SkillName, prof)
	}

	response.Created(c, map[string]string{
		"id":           id,
		"display_name": req.DisplayName,
		"tier":         "tier3",
	})
}

// List 志愿者列表
func (h *VolunteerHandler) List(c *gin.Context) {
	var query dto.VolunteerListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid params")
		return
	}
	if query.Page < 1 { query.Page = 1 }
	if query.PageSize < 1 || query.PageSize > 100 { query.PageSize = 20 }

	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if query.Tier != "" {
		where += fmt.Sprintf(" AND v.tier = $%d", argIdx)
		args = append(args, query.Tier); argIdx++
	}
	if query.Status != "" {
		where += fmt.Sprintf(" AND v.status = $%d", argIdx)
		args = append(args, query.Status); argIdx++
	}
	if query.AreaCode != "" {
		where += fmt.Sprintf(" AND v.area_code = $%d", argIdx)
		args = append(args, query.AreaCode); argIdx++
	}
	if query.Search != "" {
		where += fmt.Sprintf(" AND v.display_name ILIKE $%d", argIdx)
		args = append(args, "%"+query.Search+"%"); argIdx++
	}

	var total int64
	database.Pool.QueryRow(c, fmt.Sprintf("SELECT COUNT(*) FROM volunteers v %s", where), args...).Scan(&total)

	offset := (query.Page - 1) * query.PageSize
	dataSQL := fmt.Sprintf(`
		SELECT v.id, v.user_id, v.display_name, v.tier::text, v.trust_score, v.status::text,
			v.id_verified, COALESCE(v.cert_no, ''), COALESCE(v.area_code, ''),
			COALESCE(a.name, ''), COALESCE(v.lat, 0), COALESCE(v.lng, 0),
			v.completed_tasks, v.rating, v.abandon_count, v.total_accepted,
			v.created_at, v.updated_at
		FROM volunteers v
		LEFT JOIN areas a ON v.area_code = a.code
		%s
		ORDER BY v.trust_score DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, query.PageSize, offset)

	rows, err := database.Pool.Query(c, dataSQL, args...)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	var items []dto.VolunteerResponse
	for rows.Next() {
		var v dto.VolunteerResponse
		var lat, lng, rating float64
		var createdAt, updatedAt time.Time

		rows.Scan(&v.ID, &v.UserID, &v.DisplayName, &v.Tier, &v.TrustScore, &v.Status,
			&v.IDVerified, &v.CertNo, &v.AreaCode, &v.AreaName, &lat, &lng,
			&v.CompletedTasks, &rating, &v.AbandonCount, &v.TotalAccepted,
			&createdAt, &updatedAt)

		v.Lat = lat; v.Lng = lng; v.Rating = rating
		v.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, v)
	}

	response.Paginated(c, items, total, query.Page, query.PageSize)
}

// Get 志愿者详情
func (h *VolunteerHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var v dto.VolunteerResponse
	var lat, lng, rating float64
	var createdAt, updatedAt time.Time
	var frozenUntil *time.Time

	err := database.Pool.QueryRow(c, `
		SELECT v.id, v.user_id, v.display_name, v.tier::text, v.trust_score, v.status::text,
			v.id_verified, COALESCE(v.cert_no, ''), COALESCE(v.area_code, ''),
			COALESCE(a.name, ''), COALESCE(v.lat, 0), COALESCE(v.lng, 0),
			v.completed_tasks, v.rating, v.abandon_count, v.total_accepted,
			v.frozen_until, v.created_at, v.updated_at
		FROM volunteers v
		LEFT JOIN areas a ON v.area_code = a.code
		WHERE v.id = $1
	`, id).Scan(
		&v.ID, &v.UserID, &v.DisplayName, &v.Tier, &v.TrustScore, &v.Status,
		&v.IDVerified, &v.CertNo, &v.AreaCode, &v.AreaName, &lat, &lng,
		&v.CompletedTasks, &rating, &v.AbandonCount, &v.TotalAccepted,
		&frozenUntil, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "volunteer not found")
			return
		}
		response.InternalError(c, "query failed")
		return
	}

	v.Lat = lat; v.Lng = lng; v.Rating = rating
	v.CreatedAt = createdAt.Format(time.RFC3339)
	if frozenUntil != nil { v.FrozenUntil = frozenUntil.Format(time.RFC3339) }

	// 获取技能
	skillRows, _ := database.Pool.Query(c,
		`SELECT skill_name, proficiency FROM volunteer_skills WHERE volunteer_id = $1`, id)
	if skillRows != nil {
		defer skillRows.Close()
		for skillRows.Next() {
			var s dto.VolunteerSkillItem
			skillRows.Scan(&s.SkillName, &s.Proficiency)
			v.Skills = append(v.Skills, s)
		}
	}

	response.Success(c, v)
}

// Update 更新志愿者信息
func (h *VolunteerHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateVolunteerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	setClauses := "updated_at = NOW()"
	args := []interface{}{id}
	argIdx := 2

	if req.DisplayName != nil {
		setClauses += fmt.Sprintf(", display_name = $%d", argIdx)
		args = append(args, *req.DisplayName); argIdx++
	}
	if req.Bio != nil {
		setClauses += fmt.Sprintf(", bio = $%d", argIdx)
		args = append(args, *req.Bio); argIdx++
	}
	if req.AreaCode != nil {
		setClauses += fmt.Sprintf(", area_code = $%d", argIdx)
		args = append(args, *req.AreaCode); argIdx++
	}
	if req.Lat != nil {
		setClauses += fmt.Sprintf(", lat = $%d", argIdx)
		args = append(args, *req.Lat); argIdx++
	}
	if req.Lng != nil {
		setClauses += fmt.Sprintf(", lng = $%d", argIdx)
		args = append(args, *req.Lng); argIdx++
	}
	if req.Status != nil {
		setClauses += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, *req.Status); argIdx++
	}

	_, err := database.Pool.Exec(c, fmt.Sprintf(
		"UPDATE volunteers SET %s WHERE id = $1", setClauses), args...)
	if err != nil {
		response.InternalError(c, "update failed")
		return
	}

	response.Success(c, nil)
}
