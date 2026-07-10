package handler

import (
	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/internal/websocket"
	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SOSHandler SOS处理器
type SOSHandler struct {
	wsHub *websocket.Hub
}

// NewSOSHandler 创建SOS处理器
func NewSOSHandler(wsHub *websocket.Hub) *SOSHandler {
	return &SOSHandler{wsHub: wsHub}
}

// Trigger SOS触发
func (h *SOSHandler) Trigger(c *gin.Context) {
	var req dto.SOSTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	locationSource := req.LocationSource
	if locationSource == "" {
		locationSource = "gps"
	}

	userID, _ := c.Get("user_id")
	var uid *string
	if u, ok := userID.(string); ok && u != "" {
		uid = &u
	}

	id := uuid.New().String()
	_, err := database.Pool.Exec(c, `
		INSERT INTO sos_triggers (id, user_id, guest_phone, lat, lng, location_source,
			last_known_lat, last_known_lng, address, status, trigger_time, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'triggered',NOW(),NOW())
	`, id, uid, req.GuestPhone, req.Lat, req.Lng, locationSource,
		req.LastKnownLat, req.LastKnownLng, req.Address)
	if err != nil {
		response.InternalError(c, "sos trigger failed")
		return
	}

	// 通知指挥中心
	h.wsHub.SendToCommand(websocket.MsgTypeAlert, map[string]interface{}{
		"type":       "sos_triggered",
		"sos_id":     id,
		"user_id":    uid,
		"lat":        req.Lat,
		"lng":        req.Lng,
		"address":    req.Address,
		"location_source": locationSource,
	})

	response.Created(c, dto.SOSResponse{
		ID:             id,
		Status:         "triggered",
		LocationSource: locationSource,
	})
}

// Confirm SOS确认/取消
func (h *SOSHandler) Confirm(c *gin.Context) {
	id := c.Param("id")

	var req dto.SOSConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	switch req.Status {
	case "confirmed":
		_, err := database.Pool.Exec(c, `
			UPDATE sos_triggers SET status = 'confirmed', confirm_time = NOW()
			WHERE id = $1 AND status = 'triggered'
		`, id)
		if err != nil {
			response.InternalError(c, "confirm failed")
			return
		}

		// 自动创建求助
		var lat, lng float64
		var address, guestPhone *string
		database.Pool.QueryRow(c, `
			SELECT COALESCE(lat,0), COALESCE(lng,0), address, guest_phone FROM sos_triggers WHERE id = $1
		`, id).Scan(&lat, &lng, &address, &guestPhone)

		requestID := uuid.New().String()
		requestNo := "SOS" + uuid.New().String()[:8]
		addr := ""
		if address != nil { addr = *address }

		_, _ = database.Pool.Exec(c, `
			INSERT INTO help_requests (id, request_no, address, content, urgency, people_count,
				lat, lng, status, route, created_at, updated_at)
			VALUES ($1,$2,$3,'SOS紧急求助','critical',1,$4,$5,'pending','centralized',NOW(),NOW())
		`, requestID, requestNo, addr, lat, lng)

		// 关联到SOS
		_, _ = database.Pool.Exec(c, `
			UPDATE sos_triggers SET request_id = $1 WHERE id = $2
		`, requestID, id)

		response.Success(c, dto.SOSResponse{
			ID:        id,
			Status:    "confirmed",
			RequestID: requestID,
		})

	case "cancelled", "false_alarm":
		_, err := database.Pool.Exec(c, `
			UPDATE sos_triggers SET status = $1, cancel_time = NOW(),
				false_alarm = $2
			WHERE id = $3 AND status = 'triggered'
		`, req.Status, req.Status == "false_alarm", id)
		if err != nil {
			response.InternalError(c, "cancel failed")
			return
		}
		response.Success(c, dto.SOSResponse{ID: id, Status: req.Status})

	default:
		response.BadRequest(c, "invalid status")
	}
}
