package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"emergency-msystem-backend/config"
	"emergency-msystem-backend/internal/database"
	"emergency-msystem-backend/internal/redis"
	"emergency-msystem-backend/internal/router"
	"emergency-msystem-backend/internal/service"
	"emergency-msystem-backend/internal/websocket"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("========================================")
	log.Println("  广西洪涝灾害应急管理多端协同平台 v2.1")
	log.Println("  Emergency Management System Backend")
	log.Println("========================================")

	// 加载 .env 文件（优先从可执行文件同目录，其次从 config/ 目录）
	envPaths := []string{
		".env",
		filepath.Join("config", ".env"),
	}
	loaded := false
	for _, p := range envPaths {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err != nil {
				log.Printf("[WARN] Failed to load %s: %v", p, err)
			} else {
				log.Printf("[Config] Loaded env from: %s", p)
				loaded = true
				break
			}
		}
	}
	if !loaded {
		log.Println("[Config] No .env file found, using system environment variables")
	}

	// 加载配置
	cfg := config.Load()
	log.Printf("[Config] Server mode: %s, port: %s", cfg.Server.Mode, cfg.Server.Port)

	// 初始化 PostgreSQL
	dbPool, err := database.Init(cfg.DB)
	if err != nil {
		log.Fatalf("[FATAL] Database initialization failed: %v", err)
	}
	defer database.Close()
	_ = dbPool

	// 初始化 Redis
	redisClient, err := redis.Init(cfg.Redis)
	if err != nil {
		log.Printf("[WARN] Redis initialization failed: %v — continuing without cache", err)
	} else {
		defer redis.Close()
		_ = redisClient
	}

	// 初始化 WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// 初始化匹配引擎
	matchEngine := service.NewMatchEngine(cfg.Match, wsHub)
	defer matchEngine.Stop()

	// 设置路由
	r := router.Setup(cfg, matchEngine, wsHub)

	// HTTP 服务器（高并发配置）
	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// 启动服务器
	go func() {
		log.Printf("[Server] Listening on %s", srv.Addr)
		log.Printf("[Server] API Base: http://%s/api/v1", srv.Addr)
		log.Printf("[Server] WebSocket: ws://%s/ws/command | ws://%s/ws/user/:id", srv.Addr, srv.Addr)
		log.Printf("[Server] Health:  http://%s/health", srv.Addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server failed: %v", err)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[Server] Received signal: %v, shutting down...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[Server] Gracefully stopped")
}
