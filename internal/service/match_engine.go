package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"emergency-msystem-backend/config"
	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/models"
	"emergency-msystem-backend/internal/websocket"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MatchEngine 匹配引擎 v2.0
// 六维匹配：需求∩技能 × 距离 × Tier × 密度 × 优先级 × 防滥用
type MatchEngine struct {
	cfg    config.MatchConfig
	db     *pgx.Conn // 用于匹配查询的独立连接
	wsHub  *websocket.Hub
	mu     sync.RWMutex
	// 匹配任务队列 — 使用带缓冲channel实现worker pool
	matchJobs chan *matchJob
	stopCh    chan struct{}
}

type matchJob struct {
	HelpRequestID string
	Urgency       models.Urgency
	AreaCode      string
	Lat, Lng      float64
	VulnerableGroups []models.VulnerableGroup
	Needs         []string
	PeopleCount   int
}

// NewMatchEngine 创建匹配引擎
func NewMatchEngine(cfg config.MatchConfig, wsHub *websocket.Hub) *MatchEngine {
	engine := &MatchEngine{
		cfg:       cfg,
		wsHub:     wsHub,
		matchJobs: make(chan *matchJob, 100),
		stopCh:    make(chan struct{}),
	}

	// 启动 worker pool
	for i := 0; i < cfg.WorkerPoolSize; i++ {
		go engine.worker(i)
	}

	// 启动超时监控
	go engine.timeoutMonitor()

	// 启动 unreachable 检测
	go engine.unreachableMonitor()

	// 启动自动确认扫描
	go engine.autoConfirmScanner()

	return engine
}

// SetDB 设置数据库连接
func (me *MatchEngine) SetDB(conn *pgx.Conn) {
	me.db = conn
}

// GetAutoConfirmHours 获取自动确认小时数
func (me *MatchEngine) GetAutoConfirmHours() int {
	return me.cfg.AutoConfirmHours
}

// Submit 提交匹配任务
func (me *MatchEngine) Submit(helpRequestID string, urgency models.Urgency, areaCode string, lat, lng float64, vulnerableGroups []models.VulnerableGroup) {
	job := &matchJob{
		HelpRequestID:    helpRequestID,
		Urgency:          urgency,
		AreaCode:         areaCode,
		Lat:              lat,
		Lng:              lng,
		VulnerableGroups: vulnerableGroups,
	}

	select {
	case me.matchJobs <- job:
		log.Printf("[MatchEngine] Job submitted: %s urgency=%s", helpRequestID[:8], urgency)
	default:
		log.Printf("[MatchEngine] Job queue full, dropping: %s", helpRequestID[:8])
	}
}

// worker 匹配工作协程
func (me *MatchEngine) worker(id int) {
	log.Printf("[MatchEngine] Worker %d started", id)
	for {
		select {
		case job := <-me.matchJobs:
			me.processMatch(job)
		case <-me.stopCh:
			return
		}
	}
}

// processMatch 执行匹配
func (me *MatchEngine) processMatch(job *matchJob) {
	if me.db == nil || database.Pool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. 获取区域密度配置
	densityCfg, err := me.getAreaDensity(ctx, job.AreaCode)
	if err != nil {
		log.Printf("[MatchEngine] Failed to get area density: %v", err)
		return
	}

	radius := densityCfg.MatchRadiusKM
	if radius <= 0 {
		radius = me.cfg.MaxMatchRadiusKM
	}

	// 特殊群体缩小半径兜底
	if len(job.VulnerableGroups) > 0 {
		radius = math.Min(radius, 2.0)
	}

	// 2. 查询附近志愿者 (六维过滤)
	volunteers, err := me.findVolunteers(ctx, job, radius)
	if err != nil {
		log.Printf("[MatchEngine] Failed to find volunteers: %v", err)
		return
	}

	if len(volunteers) == 0 {
		// 农村区域启用接力匹配
		if densityCfg.RelayEnabled {
			me.attemptRelayMatch(ctx, job, radius, densityCfg)
		}

		// 无志愿者：触发升级
		me.escalate(ctx, job, "no_volunteers_found")
		return
	}

	// 3. 排序：trust_score 降序
	sort.Slice(volunteers, func(i, j int) bool {
		return volunteers[i].TrustScore > volunteers[j].TrustScore
	})

	// 4. 创建匹配记录（Top 3 志愿者）
	topN := 3
	if len(volunteers) < topN {
		topN = len(volunteers)
	}

	timeoutSec := me.cfg.DefaultTimeoutSecs[string(job.Urgency)]
	if timeoutSec == 0 {
		timeoutSec = 300
	}

	for i := 0; i < topN; i++ {
		vol := volunteers[i]
		match := models.Match{
			ID:             uuid.New().String(),
			MatchNo:        fmt.Sprintf("M%s", uuid.New().String()[:8]),
			RequestID:      job.HelpRequestID,
			VolunteerID:    vol.ID,
			Status:         "pending",
			RiskLevel:      job.Urgency,
			DistanceKM:     &vol.Distance,
			MatchRadiusUsed: &radius,
			TimeoutSec:     timeoutSec,
			RelayHop:       1,
			MatchTime:      time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		_, err := database.Pool.Exec(ctx, `
			INSERT INTO matches (id, match_no, request_id, volunteer_id, status, risk_level,
				distance_km, match_radius_used, timeout_sec, match_time, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			match.ID, match.MatchNo, match.RequestID, match.VolunteerID, match.Status,
			match.RiskLevel, match.DistanceKM, match.MatchRadiusUsed, match.TimeoutSec,
			match.MatchTime, match.CreatedAt, match.UpdatedAt,
		)
		if err != nil {
			log.Printf("[MatchEngine] Failed to create match: %v", err)
			continue
		}

		// 通知志愿者
		me.wsHub.SendToUser(vol.UserID, websocket.MsgTypeMatchUpdate, map[string]interface{}{
			"match_id":    match.ID,
			"request_id":  job.HelpRequestID,
			"status":      "pending",
			"distance_km": match.DistanceKM,
			"timeout_sec": match.TimeoutSec,
		})

		// 通知指挥中心
		me.wsHub.SendToCommand(websocket.MsgTypeMatchUpdate, match)
	}

	log.Printf("[MatchEngine] Matched %d volunteers for request %s", topN, job.HelpRequestID[:8])
}

// findVolunteers 六维匹配查找志愿者
func (me *MatchEngine) findVolunteers(ctx context.Context, job *matchJob, radius float64) ([]volunteerCandidate, error) {
	// 使用 Haversine 公式按距离排序
	rows, err := database.Pool.Query(ctx, `
		SELECT v.id, v.user_id, v.display_name, v.tier, v.trust_score, v.lat, v.lng,
			v.completed_tasks, v.rating, v.abandon_count, v.total_accepted, v.frozen_until,
			(6371 * acos(cos(radians($1)) * cos(radians(v.lat)) *
			 cos(radians(v.lng) - radians($2)) + sin(radians($1)) * sin(radians(v.lat)))) AS distance
		FROM volunteers v
		WHERE v.status = 'available'
			AND v.lat IS NOT NULL
			AND v.lng IS NOT NULL
			AND v.frozen_until IS NULL
			AND (6371 * acos(cos(radians($1)) * cos(radians(v.lat)) *
			     cos(radians(v.lng) - radians($2)) + sin(radians($1)) * sin(radians(v.lat))))
			    <= $3
		ORDER BY distance ASC
		LIMIT 20
	`, job.Lat, job.Lng, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []volunteerCandidate
	for rows.Next() {
		var v volunteerCandidate
		var completedTasks, abandonCount, totalAccepted int
		var lat, lng, rating float64
		var frozenUntil *time.Time
		err := rows.Scan(&v.ID, &v.UserID, &v.DisplayName, &v.Tier, &v.TrustScore,
			&lat, &lng, &completedTasks, &rating, &abandonCount, &totalAccepted, &frozenUntil, &v.Distance)
		if err != nil {
			continue
		}

		// 检查冻结
		if frozenUntil != nil && frozenUntil.After(time.Now()) {
			continue
		}

		// 防滥用：检查日限
		todayLimit := me.getDailyLimit(v.Tier)
		if me.isDailyLimitExceeded(ctx, v.ID, todayLimit) {
			continue
		}

		// 放弃率检查
		if totalAccepted > 0 {
			abandonRate := float64(abandonCount) / float64(totalAccepted)
			if abandonRate > me.cfg.AbandonRateThreshold {
				continue
			}
		}

		v.CompletedTasks = completedTasks
		v.Rating = rating

		candidates = append(candidates, v)
	}

	return candidates, nil
}

type volunteerCandidate struct {
	ID             string
	UserID         string
	DisplayName    string
	Tier           models.TrustTier
	TrustScore     int
	Distance       float64
	CompletedTasks int
	Rating         float64
}

// getAreaDensity 获取区域密度配置
func (me *MatchEngine) getAreaDensity(ctx context.Context, areaCode string) (*models.AreaDensityConfig, error) {
	row := database.Pool.QueryRow(ctx, `
		SELECT id, area_code, density, match_radius_km, relay_enabled, created_at
		FROM area_density_config
		WHERE area_code = $1
	`, areaCode)

	cfg := &models.AreaDensityConfig{}
	err := row.Scan(&cfg.ID, &cfg.AreaCode, &cfg.Density, &cfg.MatchRadiusKM, &cfg.RelayEnabled, &cfg.CreatedAt)
	if err != nil {
		// 默认城市配置
		return &models.AreaDensityConfig{
			Density:       "urban",
			MatchRadiusKM: 3.0,
			RelayEnabled:  false,
		}, nil
	}
	return cfg, nil
}

// getDailyLimit 获取每日接单上限（根据Tier动态计算）
func (me *MatchEngine) getDailyLimit(tier models.TrustTier) int {
	switch tier {
	case "tier1":
		return 10
	case "tier2":
		return 5
	case "tier3":
		return 3
	default:
		return 3
	}
}

// isDailyLimitExceeded 检查是否超过日限
func (me *MatchEngine) isDailyLimitExceeded(ctx context.Context, volunteerID string, limit int) bool {
	var acceptedToday int
	err := database.Pool.QueryRow(ctx, `
		SELECT accepted_count FROM volunteer_daily_stats
		WHERE volunteer_id = $1 AND stat_date = CURRENT_DATE
	`, volunteerID).Scan(&acceptedToday)
	if err != nil {
		return false // 无记录，未超限
	}
	return acceptedToday >= limit
}

// attemptRelayMatch 尝试接力匹配
func (me *MatchEngine) attemptRelayMatch(ctx context.Context, job *matchJob, radius float64, densityCfg *models.AreaDensityConfig) {
	log.Printf("[MatchEngine] Attempting relay match for request %s, radius=%.1fkm", job.HelpRequestID[:8], radius)

	// 扩大搜索半径进行接力匹配
	expandedRadius := radius * 2
	candidates, err := me.findVolunteers(ctx, job, expandedRadius)
	if err != nil || len(candidates) == 0 {
		me.escalate(ctx, job, "no_volunteers_even_with_relay")
		return
	}

	relayGroupID := uuid.New().String()
	for i, vol := range candidates {
		if i >= 2 { // 最多2跳接力
			break
		}

		matchID := uuid.New().String()
		timeoutSec := me.cfg.DefaultTimeoutSecs[string(job.Urgency)]

		_, err := database.Pool.Exec(ctx, `
			INSERT INTO matches (id, match_no, request_id, volunteer_id, status, risk_level,
				distance_km, match_radius_used, timeout_sec, relay_hop, relay_group_id,
				match_time, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			matchID, fmt.Sprintf("M%s", uuid.New().String()[:8]),
			job.HelpRequestID, vol.ID, "pending", job.Urgency,
			vol.Distance, expandedRadius, timeoutSec,
			i+1, relayGroupID,
			time.Now(), time.Now(), time.Now(),
		)
		if err != nil {
			log.Printf("[MatchEngine] Relay match insert error: %v", err)
			continue
		}

		// 通知指挥中心（接力匹配）
		me.wsHub.SendToCommand(websocket.MsgTypeMatchUpdate, map[string]interface{}{
			"type":          "relay_match",
			"match_id":      matchID,
			"request_id":    job.HelpRequestID,
			"relay_hop":     i + 1,
			"relay_group_id": relayGroupID,
			"volunteer":     vol.DisplayName,
		})
	}
}

// escalate 升级到指挥中心
func (me *MatchEngine) escalate(ctx context.Context, job *matchJob, reason string) {
	if me.db == nil {
		return
	}

	// 创建升级记录
	escalationID := uuid.New().String()
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO escalations (id, match_id, request_id, reason, level, action, created_at)
		SELECT $1, m.id, $2, $3, 'warning', 'pending', NOW()
		FROM matches m
		WHERE m.request_id = $2 AND m.status = 'pending'
		LIMIT 1`,
		escalationID, job.HelpRequestID, reason,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("[MatchEngine] Escalation insert error: %v", err)
	}

	// 更新求助状态
	_, _ = database.Pool.Exec(ctx, `
		UPDATE help_requests SET route = 'centralized', updated_at = NOW()
		WHERE id = $1 AND route = 'p2p'`,
		job.HelpRequestID,
	)

	// 通知指挥中心
	me.wsHub.SendToCommand(websocket.MsgTypeEscalation, map[string]interface{}{
		"request_id": job.HelpRequestID,
		"reason":     reason,
		"urgency":    job.Urgency,
	})
}

// timeoutMonitor 超时监控协程
func (me *MatchEngine) timeoutMonitor() {
	ticker := time.NewTicker(me.cfg.TimeoutCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			me.checkTimeouts()
		case <-me.stopCh:
			return
		}
	}
}

// checkTimeouts 检查超时的匹配
func (me *MatchEngine) checkTimeouts() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 更新 elapsed_sec
	_, _ = database.Pool.Exec(ctx, `
		UPDATE matches
		SET elapsed_sec = EXTRACT(EPOCH FROM (NOW() - match_time))::INT,
			updated_at = NOW()
		WHERE status = 'pending'
	`)

	// 标记超时
	_, _ = database.Pool.Exec(ctx, `
		UPDATE matches
		SET status = 'timeout', updated_at = NOW()
		WHERE status = 'pending' AND elapsed_sec >= timeout_sec
	`)

	// 超时的匹配转升级
	rows, err := database.Pool.Query(ctx, `
		SELECT DISTINCT request_id FROM matches
		WHERE status = 'timeout' AND
			request_id NOT IN (SELECT request_id FROM escalations WHERE resolved_at IS NULL)
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var requestID string
		if err := rows.Scan(&requestID); err != nil {
			continue
		}

		// 创建升级记录
		escalationID := uuid.New().String()
		_, _ = database.Pool.Exec(ctx, `
			INSERT INTO escalations (id, match_id, request_id, reason, level, action, created_at)
			SELECT $1, m.id, m.request_id, 'P2P匹配超时自动升级', 'warning', 'auto_escalated', NOW()
			FROM matches m
			WHERE m.request_id = $2 AND m.status = 'timeout'
			LIMIT 1`,
			escalationID, requestID,
		)

		// 通知指挥中心
		me.wsHub.SendToCommand(websocket.MsgTypeEscalation, map[string]interface{}{
			"request_id": requestID,
			"reason":     "timeout",
			"action":     "auto_escalated",
		})

		log.Printf("[MatchEngine] Request %s timeout escalation", requestID[:8])
	}
}

// unreachableMonitor 失联检测协程
func (me *MatchEngine) unreachableMonitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			me.checkUnreachable()
		case <-me.stopCh:
			return
		}
	}
}

// checkUnreachable 检测失联的志愿者（接单后30min GPS未移动>100m）
func (me *MatchEngine) checkUnreachable() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := database.Pool.Exec(ctx, `
		WITH recent_tracks AS (
			SELECT match_id, volunteer_id,
				ST_Distance(
					ST_MakePoint(MIN(lng), MIN(lat))::geography,
					ST_MakePoint(MAX(lng), MAX(lat))::geography
				) AS moved_meters
			FROM volunteer_location_tracks
			WHERE recorded_at >= NOW() - INTERVAL '30 minutes'
			GROUP BY match_id, volunteer_id
		)
		UPDATE matches m
		SET status = 'unreachable',
			unreachable_detected_at = NOW(),
			updated_at = NOW()
		FROM recent_tracks rt
		WHERE m.id = rt.match_id
			AND m.volunteer_id = rt.volunteer_id
			AND m.status = 'accepted'
			AND m.accept_time <= NOW() - INTERVAL '30 minutes'
			AND rt.moved_meters <= 100
	`)
	if err != nil {
		log.Printf("[MatchEngine] Unreachable check error: %v", err)
	}

	// 通知指挥中心
	me.wsHub.SendToCommand(websocket.MsgTypeAlert, map[string]interface{}{
		"type":    "unreachable_detected",
		"message": "有志愿者可能失联（30分钟GPS未移动超过100m）",
	})
}

// autoConfirmScanner 自动确认扫描（72小时）
func (me *MatchEngine) autoConfirmScanner() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			me.autoConfirm()
		case <-me.stopCh:
			return
		}
	}
}

// autoConfirm 自动确认过期任务
func (me *MatchEngine) autoConfirm() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := database.Pool.Exec(ctx, `
		UPDATE task_completions
		SET status = 'auto_confirmed',
			confirm_method = 'auto',
			confirm_time = NOW()
		WHERE status = 'submitted'
			AND auto_confirm_deadline <= NOW()
	`)
	if err != nil {
		log.Printf("[MatchEngine] Auto confirm error: %v", err)
	}
}

// Stop 停止引擎
func (me *MatchEngine) Stop() {
	close(me.stopCh)
	log.Println("[MatchEngine] Stopped")
}
