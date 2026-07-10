package handler

import (
	"fmt"
	"time"

	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NotificationHandler 通知处理器
type NotificationHandler struct{}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// ListNotifications 通知列表
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var query dto.NotificationListQuery
	c.ShouldBindQuery(&query)
	if query.Page < 1 { query.Page = 1 }
	if query.PageSize < 1 || query.PageSize > 50 { query.PageSize = 20 }

	where := fmt.Sprintf("WHERE user_id = $1")
	args := []interface{}{userID}
	argIdx := 2

	if query.ReadStatus != nil {
		where += fmt.Sprintf(" AND read_status = $%d", argIdx)
		args = append(args, *query.ReadStatus); argIdx++
	}
	if query.Urgency != "" {
		where += fmt.Sprintf(" AND urgency = $%d", argIdx)
		args = append(args, query.Urgency); argIdx++
	}

	var total int64
	database.Pool.QueryRow(c, fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", where), args...).Scan(&total)

	offset := (query.Page - 1) * query.PageSize
	rows, err := database.Pool.Query(c, fmt.Sprintf(`
		SELECT id, title, body, urgency::text, action_type, action_data,
			read_status, created_at, read_at
		FROM notifications
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1), append(args, query.PageSize, offset)...)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	var items []dto.NotificationResponse
	for rows.Next() {
		var n dto.NotificationResponse
		var actionType *string
		var actionData []byte
		var createdAt time.Time
		var readAt *time.Time

		rows.Scan(&n.ID, &n.Title, &n.Body, &n.Urgency, &actionType, &actionData,
			&n.ReadStatus, &createdAt, &readAt)
		if actionType != nil { n.ActionType = *actionType }
		n.CreatedAt = createdAt
		n.ReadAt = readAt
		items = append(items, n)
	}

	response.Paginated(c, items, total, query.Page, query.PageSize)
}

// MarkRead 标记已读
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	_, err := database.Pool.Exec(c, `
		UPDATE notifications SET read_status = TRUE, read_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		response.InternalError(c, "mark read failed")
		return
	}

	response.Success(c, nil)
}

// MarkAllRead 全部已读
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	_, err := database.Pool.Exec(c, `
		UPDATE notifications SET read_status = TRUE, read_at = NOW()
		WHERE user_id = $1 AND read_status = FALSE
	`, userID)
	if err != nil {
		response.InternalError(c, "mark all read failed")
		return
	}

	response.Success(c, nil)
}

// Subscribe 订阅通知
func (h *NotificationHandler) Subscribe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req dto.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	templateIDs := req.GetTemplateIDs()
	if len(templateIDs) == 0 {
		response.BadRequest(c, "template_id or template_ids is required")
		return
	}

	subscribeType := req.GetSubscribeType()
	if subscribeType == "" {
		subscribeType = "longterm"
	}

	for _, tid := range templateIDs {
		// 查找模板
		var templateDBID string
		err := database.Pool.QueryRow(c, `
			SELECT id FROM notification_templates WHERE template_id = $1 AND status = 'active'
		`, tid).Scan(&templateDBID)
		if err != nil {
			continue // 跳过不存在的模板
		}

		// 创建或更新订阅
		subID := uuid.New().String()
		_, _ = database.Pool.Exec(c, `
			INSERT INTO subscriptions (id, user_id, template_id, type, status, subscribed_at)
			VALUES ($1,$2,$3,$4,'subscribed',NOW())
			ON CONFLICT (user_id, template_id)
			DO UPDATE SET status = 'subscribed', type = $4, subscribed_at = NOW()
		`, subID, userID, templateDBID, subscribeType)
	}

	response.Created(c, nil)
}

// GetPreferences 获取通知偏好
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := database.Pool.Query(c, `
		SELECT nt.template_id, nt.title, np.enabled
		FROM notification_preferences np
		JOIN notification_templates nt ON np.template_id = nt.id
		WHERE np.user_id = $1
		ORDER BY nt.title
	`, userID)
	if err != nil {
		response.InternalError(c, "query failed")
		return
	}
	defer rows.Close()

	var prefs []dto.PreferenceResponse
	for rows.Next() {
		var p dto.PreferenceResponse
		rows.Scan(&p.TemplateID, &p.TemplateName, &p.Enabled)
		prefs = append(prefs, p)
	}

	response.Success(c, prefs)
}

// UpdatePreference 更新通知偏好
func (h *NotificationHandler) UpdatePreference(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req dto.UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	// 查找模板的数据库 ID
	var templateDBID string
	err := database.Pool.QueryRow(c, `
		SELECT id FROM notification_templates WHERE template_id = $1 AND status = 'active'
	`, req.TemplateID).Scan(&templateDBID)
	if err != nil {
		response.NotFound(c, "template not found")
		return
	}

	_, err = database.Pool.Exec(c, `
		INSERT INTO notification_preferences (id, user_id, template_id, enabled, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, template_id)
		DO UPDATE SET enabled = $4, updated_at = NOW()
	`, uuid.New().String(), userID, templateDBID, req.Enabled)
	if err != nil {
		response.InternalError(c, "update preference failed")
		return
	}

	response.Success(c, nil)
}
