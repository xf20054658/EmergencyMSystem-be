package service

import (
	"fmt"
	"math/rand"
	"time"

	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/dto"
	"emergency-msystem-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CommunicationService 隐私通话服务（AXB虚拟号码 + VoIP）
type CommunicationService struct{}

// NewCommunicationService 创建通话服务
func NewCommunicationService() *CommunicationService {
	return &CommunicationService{}
}

// virtualPhonePool 模拟虚拟号码池（生产环境对接阿里云/腾讯云号码隐私保护API）
var virtualPhonePool = []string{
	"4008201100",
	"4008201101",
	"4008201102",
	"4008201103",
	"4008201104",
	"4008201105",
	"4008201106",
	"4008201107",
	"4008201108",
	"4008201109",
}

// BindPhone 创建 AXB 虚拟号码绑定
// A(求助方) → X(虚拟号) → B(支援方)，双方拨打 X 即可接通对方
func (s *CommunicationService) BindPhone(c *gin.Context, matchID, provider string) (*dto.VirtualPhoneResponse, error) {
	// 1. 检查匹配是否存在且状态为 accepted
	var matchStatus string
	err := database.Pool.QueryRow(c,
		`SELECT status::text FROM matches WHERE id = $1`, matchID,
	).Scan(&matchStatus)
	if err != nil {
		return nil, fmt.Errorf("匹配不存在")
	}
	if matchStatus != "accepted" && matchStatus != "enroute" && matchStatus != "waiting" {
		return nil, fmt.Errorf("当前匹配状态(%s)不支持绑定虚拟号码", matchStatus)
	}

	// 2. 检查是否已有有效绑定
	var existingCount int
	database.Pool.QueryRow(c,
		`SELECT COUNT(*) FROM virtual_phone_bindings WHERE match_id = $1 AND status = 'bound' AND expire_time > NOW()`,
		matchID,
	).Scan(&existingCount)
	if existingCount > 0 {
		return nil, fmt.Errorf("该匹配已有有效绑定，请先解绑")
	}

	// 3. 从号码池选取虚拟号（避免同一时间重复）
	virtualPhone := virtualPhonePool[rand.Intn(len(virtualPhonePool))]
	// 添加随机后缀区分
	virtualPhone = virtualPhone + fmt.Sprintf("%02d", rand.Intn(100))

	// 4. 模拟获取求助方和支援方手机号（生产环境从真实号码加密存储）
	// 这里使用脱敏号码，生产环境对接运营商 API 完成 AXB 绑定
	callerPhoneEncrypted := []byte("encrypted_caller_phone")
	calleePhoneEncrypted := []byte("encrypted_callee_phone")

	bindID := uuid.New().String()
	now := time.Now()
	expireTime := now.Add(24 * time.Hour)

	// 5. 插入绑定记录
	_, err = database.Pool.Exec(c, `
		INSERT INTO virtual_phone_bindings
			(id, match_id, provider, provider_bind_id, virtual_phone,
			 caller_phone_encrypted, callee_phone_encrypted,
			 status, bind_time, expire_time, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'bound',$8,$9,NOW())
	`, bindID, matchID, provider, "mock_bind_"+bindID[:8], virtualPhone,
		callerPhoneEncrypted, calleePhoneEncrypted,
		now, expireTime)
	if err != nil {
		return nil, fmt.Errorf("创建绑定失败: %v", err)
	}

	// 6. 更新 matches 表的 virtual_phone 字段
	_, _ = database.Pool.Exec(c,
		`UPDATE matches SET virtual_phone = $1, updated_at = NOW() WHERE id = $2`,
		virtualPhone, matchID)

	return &dto.VirtualPhoneResponse{
		ID:           bindID,
		VirtualPhone: virtualPhone,
		Status:       "bound",
		ExpireTime:   expireTime.Format(time.RFC3339),
	}, nil
}

// Call 发起通话（模拟运营商回拨）
func (s *CommunicationService) Call(c *gin.Context, matchID, callType, callerRole string) (*dto.CallRecordResponse, error) {
	// 1. 验证绑定存在
	var bindingID, bindingStatus string
	err := database.Pool.QueryRow(c,
		`SELECT id, status::text FROM virtual_phone_bindings
		 WHERE match_id = $1 AND status IN ('bound','in_call') AND expire_time > NOW()
		 ORDER BY bind_time DESC LIMIT 1`,
		matchID,
	).Scan(&bindingID, &bindingStatus)
	if err != nil {
		return nil, fmt.Errorf("没有有效的虚拟号码绑定，请先绑定")
	}

	// 2. 创建通话记录
	callID := uuid.New().String()
	startTime := time.Now()

	// 模拟通话时长（5-180秒，voip略短）
	duration := rand.Intn(120) + 5
	if callType == "phone" {
		duration = rand.Intn(175) + 5
	}

	status := "in_progress"
	if rand.Intn(100) < 85 { // 85% 成功率
		status = "success"
	} else {
		status = "no_answer"
		duration = 0
	}

	endTime := startTime.Add(time.Duration(duration) * time.Second)

	_, err = database.Pool.Exec(c, `
		INSERT INTO call_records
			(id, match_id, binding_id, call_type, caller_role,
			 caller_phone_encrypted, duration_sec, status,
			 start_time, end_time, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
	`, callID, matchID, bindingID, models.CallType(callType), models.CallerRole(callerRole),
		[]byte("encrypted_caller"), duration, models.CallStatus(status),
		startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("创建通话记录失败: %v", err)
	}

	// 3. 更新绑定状态为 in_call
	if bindingStatus == "bound" {
		_, _ = database.Pool.Exec(c,
			`UPDATE virtual_phone_bindings SET status = 'in_call', updated_at = NOW() WHERE id = $1`,
			bindingID)
	}

	endTimeStr := endTime.Format(time.RFC3339)
	if duration == 0 {
		endTimeStr = ""
	}

	return &dto.CallRecordResponse{
		ID:          callID,
		MatchID:     matchID,
		CallType:    callType,
		CallerRole:  callerRole,
		DurationSec: duration,
		Status:      status,
		StartTime:   startTime.Format(time.RFC3339),
		EndTime:     endTimeStr,
	}, nil
}

// Unbind 解绑虚拟号码
func (s *CommunicationService) Unbind(c *gin.Context, matchID string) error {
	now := time.Now()
	result, err := database.Pool.Exec(c, `
		UPDATE virtual_phone_bindings
		SET status = 'unbound', unbind_time = $1, updated_at = $1
		WHERE match_id = $2 AND status IN ('bound','in_call') AND expire_time > NOW()
	`, now, matchID)
	if err != nil {
		return fmt.Errorf("解绑失败: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("没有有效的绑定可以解绑")
	}

	// 清除 matches 表的 virtual_phone 字段
	_, _ = database.Pool.Exec(c,
		`UPDATE matches SET virtual_phone = NULL, updated_at = NOW() WHERE id = $1`,
		matchID)

	return nil
}

// GetBinding 获取当前绑定信息
func (s *CommunicationService) GetBinding(c *gin.Context, matchID string) (*dto.VirtualPhoneResponse, error) {
	var resp dto.VirtualPhoneResponse
	var bindTime, expireTime time.Time
	var bindingStatus models.BindingStatus

	err := database.Pool.QueryRow(c, `
		SELECT id, virtual_phone, status::text, bind_time, expire_time
		FROM virtual_phone_bindings
		WHERE match_id = $1 AND status IN ('bound','in_call') AND expire_time > NOW()
		ORDER BY bind_time DESC LIMIT 1
	`, matchID).Scan(&resp.ID, &resp.VirtualPhone, &bindingStatus, &bindTime, &expireTime)
	if err != nil {
		return nil, fmt.Errorf("未找到有效绑定")
	}

	resp.Status = string(bindingStatus)
	resp.ExpireTime = expireTime.Format(time.RFC3339)
	return &resp, nil
}

// GetCallRecords 获取通话记录列表
func (s *CommunicationService) GetCallRecords(c *gin.Context, matchID string) ([]dto.CallRecordResponse, error) {
	rows, err := database.Pool.Query(c, `
		SELECT id, match_id, call_type::text, caller_role::text,
			duration_sec, status::text, start_time, end_time
		FROM call_records
		WHERE match_id = $1
		ORDER BY start_time DESC
		LIMIT 50
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("查询通话记录失败: %v", err)
	}
	defer rows.Close()

	var records []dto.CallRecordResponse
	for rows.Next() {
		var r dto.CallRecordResponse
		var startTime, endTime *time.Time
		rows.Scan(&r.ID, &r.MatchID, &r.CallType, &r.CallerRole,
			&r.DurationSec, &r.Status, &startTime, &endTime)
		if startTime != nil {
			r.StartTime = startTime.Format(time.RFC3339)
		}
		if endTime != nil {
			r.EndTime = endTime.Format(time.RFC3339)
		}
		records = append(records, r)
	}
	if records == nil {
		records = []dto.CallRecordResponse{}
	}
	return records, nil
}
