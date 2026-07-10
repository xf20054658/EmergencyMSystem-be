package router

import (
	"emergency-msystem-backend/config"
	"emergency-msystem-backend/internal/handler"
	"emergency-msystem-backend/internal/middleware"
	"emergency-msystem-backend/internal/service"
	"emergency-msystem-backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

// Setup 配置路由
func Setup(cfg *config.Config, engine *service.MatchEngine, wsHub *websocket.Hub) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORSMiddleware(cfg.CORS))

	// 限流
	limiter := middleware.NewRateLimiter(cfg.RateLimit.RequestsPerMin)
	if cfg.RateLimit.Enabled {
		r.Use(middleware.RateLimit(limiter))
	}

	// 创建 Handler
	authSvc := service.NewAuthService(*cfg)
	authH := handler.NewAuthHandler(authSvc, *cfg)
	helpReqH := handler.NewHelpRequestHandler(engine, wsHub)
	matchH := handler.NewMatchHandler(engine, wsHub)
	volunteerH := handler.NewVolunteerHandler(wsHub)
	sosH := handler.NewSOSHandler(wsHub)
	notifH := handler.NewNotificationHandler()
	dashboardH := handler.NewDashboardHandler()
	wsH := handler.NewWebSocketHandler(wsHub)

	// --- WebSocket ---
	r.GET("/ws/command", middleware.AuthRequired(cfg.JWT), wsH.CommandWS)
	r.GET("/ws/user/:id", middleware.AuthRequired(cfg.JWT), wsH.UserWS)

	// --- API v1 ---
	v1 := r.Group("/api/v1")

	// 认证模块（无需鉴权）
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authH.Login)
		auth.POST("/login/phone", authH.PhoneLogin)
	}

	// 需要鉴权的路由
	authorized := v1.Group("")
	authorized.Use(middleware.AuthRequired(cfg.JWT))

	// 用户
	authorized.GET("/user/profile", authH.GetProfile)
	authorized.PUT("/user/location", authH.UpdateLocation)

	// 求助模块
	helpRequests := authorized.Group("/help-requests")
	{
		helpRequests.POST("", helpReqH.Create)
		helpRequests.GET("", helpReqH.List)
		helpRequests.GET("/:id", helpReqH.Get)
		helpRequests.PUT("/:id", helpReqH.Update)
		helpRequests.POST("/:id/cancel", helpReqH.Cancel)
	}

	// 匹配模块
	matches := authorized.Group("/matches")
	{
		matches.GET("", matchH.List)
		matches.GET("/:id", matchH.Get)
		matches.PUT("/:id/accept", middleware.VolunteerRequired(), matchH.Accept)
		matches.PUT("/:id/reject", middleware.VolunteerRequired(), matchH.Reject)
		matches.PUT("/:id/enroute", middleware.VolunteerRequired(), matchH.SetEnroute)
		matches.PUT("/:id/waiting", middleware.VolunteerRequired(), matchH.SetWaiting)
		matches.POST("/:id/complete", middleware.VolunteerRequired(), matchH.Complete)
		matches.POST("/:id/confirm", matchH.ConfirmCompletion)
		matches.POST("/:id/reinforce", middleware.VolunteerRequired(), matchH.Reinforce)
	}

	// GPS上报
	authorized.POST("/providers/gps", middleware.VolunteerRequired(), matchH.GPSReport)

	// 志愿者模块
	providers := authorized.Group("/providers")
	{
		providers.POST("/register", volunteerH.Register)
		providers.GET("", volunteerH.List)
		providers.GET("/:id", volunteerH.Get)
		providers.PUT("/:id", volunteerH.Update)
	}

	// SOS模块
	sos := authorized.Group("/sos")
	{
		sos.POST("/trigger", sosH.Trigger)
		sos.PUT("/:id/confirm", sosH.Confirm)
	}

	// 通知模块
	notifications := authorized.Group("/notifications")
	{
		notifications.GET("", notifH.ListNotifications)
		notifications.PUT("/:id/read", notifH.MarkRead)
		notifications.PUT("/read-all", notifH.MarkAllRead)
		notifications.POST("/subscribe", notifH.Subscribe)
		notifications.GET("/preferences", notifH.GetPreferences)
		notifications.PUT("/preferences", notifH.UpdatePreference)
	}

	// 指挥中心
	command := authorized.Group("")
	{
		command.GET("/dashboard/kpi", dashboardH.GetKPI)
		command.GET("/dashboard/urgency-distribution", dashboardH.GetUrgencyDistribution)
		command.GET("/dashboard/match-stats", dashboardH.GetMatchStats)
		command.GET("/shelters", dashboardH.GetShelters)
		command.GET("/resources", dashboardH.GetResources)
		command.GET("/rescue-teams", dashboardH.GetRescueTeams)
		command.GET("/escalations", dashboardH.GetEscalations)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "emergency-msystem"})
	})

	return r
}
