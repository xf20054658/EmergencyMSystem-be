package config

import (
	"os"
	"strconv"
	"time"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	CORS     CORSConfig
	Match    MatchConfig
	RateLimit RateLimitConfig
	Log      LogConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Mode         string // debug / release / test
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func (c DBConfig) DSN() string {
	return "postgres://" + c.User + ":" + c.Password +
		"@" + c.Host + ":" + c.Port +
		"/" + c.DBName + "?sslmode=" + c.SSLMode
}

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type JWTConfig struct {
	Secret            string
	AccessTokenExpire time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type MatchConfig struct {
	WorkerPoolSize     int           // 匹配工作协程池大小
	TimeoutCheckInterval time.Duration // 超时检查间隔
	GPSReportInterval    time.Duration // GPS上报间隔
	MaxMatchRadiusKM     float64       // 最大匹配半径
	DefaultTimeoutSecs   map[string]int // urgency -> timeout secs
	AbandonRateThreshold float64       // 放弃率冻结阈值
	FrozenDuration       time.Duration // 冻结时长
	AutoConfirmHours     int           // 自动确认小时数
}

type RateLimitConfig struct {
	Enabled      bool
	RequestsPerMin int
}

type LogConfig struct {
	Level  string // debug / info / warn / error
	Format string // json / text
}

// Load 从环境变量加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  mustParseDuration(getEnv("SERVER_READ_TIMEOUT", "30s")),
			WriteTimeout: mustParseDuration(getEnv("SERVER_WRITE_TIMEOUT", "30s")),
			IdleTimeout:  mustParseDuration(getEnv("SERVER_IDLE_TIMEOUT", "120s")),
			Mode:         getEnv("GIN_MODE", "debug"),
		},
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "emergency_msystem"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    mustAtoi(getEnv("DB_MAX_OPEN_CONNS", "50")),
			MaxIdleConns:    mustAtoi(getEnv("DB_MAX_IDLE_CONNS", "10")),
			ConnMaxLifetime: mustParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "30m")),
			ConnMaxIdleTime: mustParseDuration(getEnv("DB_CONN_MAX_IDLE_TIME", "5m")),
		},
		Redis: RedisConfig{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           mustAtoi(getEnv("REDIS_DB", "0")),
			PoolSize:     mustAtoi(getEnv("REDIS_POOL_SIZE", "100")),
			MinIdleConns: mustAtoi(getEnv("REDIS_MIN_IDLE_CONNS", "10")),
			DialTimeout:  mustParseDuration(getEnv("REDIS_DIAL_TIMEOUT", "5s")),
			ReadTimeout:  mustParseDuration(getEnv("REDIS_READ_TIMEOUT", "3s")),
			WriteTimeout: mustParseDuration(getEnv("REDIS_WRITE_TIMEOUT", "3s")),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "emergency-msystem-secret-key-change-in-production"),
			AccessTokenExpire: mustParseDuration(getEnv("JWT_ACCESS_EXPIRE", "168h")), // 7天
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		},
		Match: MatchConfig{
			WorkerPoolSize:      mustAtoi(getEnv("MATCH_WORKER_POOL_SIZE", "20")),
			TimeoutCheckInterval: mustParseDuration(getEnv("MATCH_TIMEOUT_CHECK_INTERVAL", "10s")),
			GPSReportInterval:    mustParseDuration(getEnv("GPS_REPORT_INTERVAL", "15s")),
			MaxMatchRadiusKM:     mustParseFloat(getEnv("MATCH_MAX_RADIUS_KM", "10")),
			DefaultTimeoutSecs: map[string]int{
				"critical": 120,
				"high":     180,
				"medium":   300,
				"low":      600,
			},
			AbandonRateThreshold: 0.30,
			FrozenDuration:       24 * time.Hour,
			AutoConfirmHours:     72,
		},
		RateLimit: RateLimitConfig{
			Enabled:       getEnv("RATE_LIMIT_ENABLED", "true") == "true",
			RequestsPerMin: mustAtoi(getEnv("RATE_LIMIT_RPM", "300")),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustAtoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func mustParseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

func mustParseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
