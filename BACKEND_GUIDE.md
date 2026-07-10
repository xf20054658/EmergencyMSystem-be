# 后端开发指南 (Go + Gin 高并发架构)

## 1. 技术栈

| 层级 | 技术选型 | 版本 | 说明 |
|------|---------|------|------|
| **语言** | Go | 1.21+ | 高并发、低内存 |
| **Web 框架** | Gin | v1.9+ | 高性能 HTTP 路由器 |
| **数据库** | PostgreSQL | 14+ | pgxpool 连接池 |
| **缓存** | Redis | 7+ | go-redis 客户端 |
| **WebSocket** | Gorilla WebSocket | v1.5+ | 实时双向通信 |
| **JWT** | golang-jwt | v5 | 无状态鉴权 |
| **加密** | crypto/aes | - | AES-256-GCM 手机号加密 |
| **匹配引擎** | 自研 | - | 六维匹配算法 Worker Pool |
| **限流** | 自研令牌桶 | - | 无外部依赖 |

## 2. 项目结构

```
src/backend/
├── main.go                      # 入口：服务启动、优雅退出
├── config/
│   ├── config.go                # 配置加载（环境变量）
│   └── .env                     # 环境变量示例
├── internal/
│   ├── database/
│   │   └── pg.go                # PostgreSQL pgxpool 连接池
│   ├── redis/
│   │   └── client.go            # Redis 连接管理
│   ├── models/
│   │   └── models.go            # 33个领域模型（匹配数据库DDL）
│   ├── dto/
│   │   ├── auth.go              # 认证 DTO
│   │   ├── help_request.go      # 求助 DTO
│   │   ├── match.go             # 匹配 DTO
│   │   ├── volunteer.go         # 志愿者 DTO
│   │   ├── notification.go      # 通知 DTO
│   │   ├── communication.go     # 通话 DTO
│   │   ├── review.go            # 评价 DTO
│   │   ├── sos.go               # SOS DTO
│   │   ├── common.go            # 通用 DTO
│   │   └── dashboard.go         # 仪表盘 DTO
│   ├── handler/
│   │   ├── auth.go              # 认证接口
│   │   ├── help_request.go      # 求助 CRUD
│   │   ├── match.go             # 匹配生命周期
│   │   ├── volunteer.go         # 志愿者管理
│   │   ├── sos.go               # SOS 触发/确认
│   │   ├── notification.go      # 通知管理
│   │   ├── dashboard.go         # 指挥中心仪表盘
│   │   └── websocket_handler.go # WebSocket 连接
│   ├── service/
│   │   ├── match_engine.go      # 六维匹配引擎 (Worker Pool)
│   │   └── auth_service.go      # 认证服务
│   ├── middleware/
│   │   ├── auth.go              # JWT 鉴权、角色校验、CORS
│   │   ├── logger.go            # 请求日志 + Recovery
│   │   └── ratelimit.go         # 令牌桶限流
│   ├── websocket/
│   │   └── hub.go               # WebSocket Hub 连接管理
│   └── router/
│       └── router.go            # 路由注册
├── pkg/
│   ├── jwt/
│   │   └── jwt.go               # JWT 生成/解析
│   ├── crypto/
│   │   └── crypto.go            # AES加密、脱敏、哈希
│   └── response/
│       └── response.go          # 统一响应格式
├── go.mod
├── go.sum
└── BACKEND_GUIDE.md
```

## 3. 核心架构设计

### 3.1 高并发设计

#### Worker Pool 匹配引擎
```go
// MatchEngine 使用 Worker Pool 模式处理匹配任务
type MatchEngine struct {
    matchJobs chan *matchJob     // 缓冲 100 的 Job Queue
    stopCh    chan struct{}
    // 启动 cfg.WorkerPoolSize (默认20) 个 Worker
}
```

#### PostgreSQL 连接池
```go
// pgxpool 连接池配置
MaxConns:        50    // 最大连接数
MinConns:        10    // 最小空闲连接
ConnMaxLifetime: 30m   // 连接最大存活时间
ConnMaxIdleTime: 5m    // 空闲连接超时
```

#### Redis 连接池
```go
PoolSize:     100   // 连接池大小
MinIdleConns: 10    // 最小空闲连接
```

### 3.2 六维匹配算法 (v2.0)

```
Score = (需求∩技能 × 0.30) × (距离分 × 0.25) × (Tier分 × 0.20)
      × (密度因子 × 0.10) × (优先级 × 0.10) × (防滥用 × 0.05)
```

| 维度 | 权重 | 说明 |
|------|------|------|
| 需求∩技能 | 30% | 支持方的技能与求助物资需求的重合度 |
| 距离分 | 25% | Haversine 公式计算，加权降序 |
| Tier分 | 20% | tier1(100) > tier2(70) > tier3(40) |
| 密度因子 | 10% | 城市半径3km/郊区5km/农村10km |
| 优先级 | 10% | critical > high > medium > low |
| 防滥用 | 5% | 放弃率>30%冻结、日接单上限 |

### 3.3 匹配生命周期

```
pending → accepted → enroute → waiting(可逆) → completed
  ↓         ↓           ↓
rejected  timeout    unreachable(30min GPS未移动>100m)
  ↓                   ↓
escalated ←──────────┘
```

### 3.4 WebSocket 实时通信

```
Mobile端 (<->) WebSocket (/ws/user/:id)  → Hub.userClients
PC指挥中心 (<->) WebSocket (/ws/command) → Hub.commandClients

消息类型：
- new_help_request  → 新求助
- match_update      → 匹配状态变更
- gps_update        → 志愿者GPS上报
- alert             → 告警
- escalation        → 升级通知
- system_broadcast  → 系统广播
```

### 3.5 限流策略

```
令牌桶算法：
- 全局限流：300 RPM (请求/分钟)
- 按 ClientIP/UserID 分组
- 5分钟未活跃自动清理
```

## 4. API 路由

### 4.1 认证模块
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/auth/login` | 否 | 微信登录 |
| POST | `/api/v1/auth/login/phone` | 否 | 手机号验证码登录 |
| GET | `/api/v1/user/profile` | 是 | 获取用户信息 |
| PUT | `/api/v1/user/location` | 是 | 更新用户位置 |

### 4.2 求助模块
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/help-requests` | 是 | 创建求助 |
| GET | `/api/v1/help-requests` | 是 | 求助列表（分页+筛选） |
| GET | `/api/v1/help-requests/:id` | 是 | 求助详情（含需求+匹配） |
| PUT | `/api/v1/help-requests/:id` | 是 | 更新求助 |
| POST | `/api/v1/help-requests/:id/cancel` | 是 | 取消求助 |

### 4.3 匹配模块
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/v1/matches` | 是 | 匹配列表 |
| GET | `/api/v1/matches/:id` | 是 | 匹配详情 |
| PUT | `/api/v1/matches/:id/accept` | 志愿者 | 接受匹配 |
| PUT | `/api/v1/matches/:id/reject` | 志愿者 | 拒绝匹配 |
| PUT | `/api/v1/matches/:id/enroute` | 志愿者 | 设为前往中 |
| PUT | `/api/v1/matches/:id/waiting` | 志愿者 | 设为等待中 |
| POST | `/api/v1/matches/:id/complete` | 志愿者 | 提交完成 |
| POST | `/api/v1/matches/:id/confirm` | 是 | 确认完成 |
| POST | `/api/v1/matches/:id/reinforce` | 志愿者 | 请求增援 |
| POST | `/api/v1/providers/gps` | 志愿者 | GPS 上报 |

### 4.4 志愿者模块
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/providers/register` | 是 | 注册成为志愿者 |
| GET | `/api/v1/providers` | 是 | 志愿者列表 |
| GET | `/api/v1/providers/:id` | 是 | 志愿者详情（含技能） |
| PUT | `/api/v1/providers/:id` | 是 | 更新志愿者信息 |

### 4.5 SOS模块
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/sos/trigger` | 是 | 触发SOS |
| PUT | `/api/v1/sos/:id/confirm` | 是 | 确认/取消SOS |

### 4.6 通知模块
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/v1/notifications` | 是 | 通知列表 |
| PUT | `/api/v1/notifications/:id/read` | 是 | 标记已读 |
| PUT | `/api/v1/notifications/read-all` | 是 | 全部已读 |
| POST | `/api/v1/notifications/subscribe` | 是 | 订阅通知 |
| GET | `/api/v1/notifications/preferences` | 是 | 通知偏好 |
| PUT | `/api/v1/notifications/preferences` | 是 | 更新偏好 |

### 4.7 指挥中心
| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/v1/dashboard/kpi` | 是 | 仪表盘KPI |
| GET | `/api/v1/dashboard/urgency-distribution` | 是 | 紧急程度分布 |
| GET | `/api/v1/dashboard/match-stats` | 是 | 匹配效率统计 |
| GET | `/api/v1/shelters` | 是 | 安置点列表 |
| GET | `/api/v1/resources` | 是 | 物资列表 |
| GET | `/api/v1/rescue-teams` | 是 | 救援队伍列表 |
| GET | `/api/v1/escalations` | 是 | 升级告警列表 |

### 4.8 WebSocket
| 路径 | 鉴权 | 说明 |
|------|------|------|
| `ws://host/ws/command` | 是 | 指挥中心实时通道 |
| `ws://host/ws/user/:id` | 是 | 用户端实时通道 |

## 5. 数据库

### 5.1 连接信息
```
Host:     localhost:5432
Database: emergency_msystem
User:     postgres
SSL:      disable (开发环境)
```

### 5.2 迁移脚本
完整的 33 表 DDL 位于 `docs/html/designer/database/schema.sql`
包含所有索引、触发器、视图。

## 6. 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_HOST` | `0.0.0.0` | 监听地址 |
| `SERVER_PORT` | `8080` | 监听端口 |
| `GIN_MODE` | `debug` | debug/release/test |
| `DB_HOST` | `localhost` | 数据库主机 |
| `DB_PORT` | `5432` | 数据库端口 |
| `DB_USER` | `postgres` | 数据库用户 |
| `DB_PASSWORD` | `` | 数据库密码 |
| `DB_NAME` | `emergency_msystem` | 数据库名称 |
| `DB_MAX_OPEN_CONNS` | `50` | 最大连接数 |
| `DB_MAX_IDLE_CONNS` | `10` | 最小空闲连接 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址 |
| `REDIS_POOL_SIZE` | `100` | Redis 连接池 |
| `JWT_SECRET` | `emergency-msystem-secret-key-change-in-production` | JWT 密钥 |
| `JWT_ACCESS_EXPIRE` | `168h` | Token 过期时间 |
| `RATE_LIMIT_ENABLED` | `true` | 启用限流 |
| `RATE_LIMIT_RPM` | `300` | 每分钟请求数 |
| `MATCH_WORKER_POOL_SIZE` | `20` | 匹配 Worker 数量 |
| `MATCH_MAX_RADIUS_KM` | `10` | 最大匹配半径 |

## 7. 启动命令

```bash
# 1. 数据库迁移
psql -U postgres -d emergency_msystem -f docs/html/designer/database/schema.sql

# 2. 安装依赖
cd src/backend
go mod tidy

# 3. 启动服务
go run main.go

# 4. 生产构建
go build -o bin/server main.go
./bin/server
```

## 8. 响应格式

### 成功
```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

### 分页
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

### 错误
```json
{
  "code": 40000,
  "message": "invalid request"
}
```

### 错误码
| HTTP | code | 说明 |
|------|------|------|
| 400 | 40000 | 请求参数错误 |
| 401 | 40100 | 未授权 |
| 403 | 40300 | 无权限 |
| 404 | 40400 | 资源不存在 |
| 409 | 40900 | 资源冲突 |
| 429 | 42900 | 请求过多 |
| 500 | 50000 | 服务器内部错误 |

## 9. Agent 行为约束
- 接口返回值必须与 `docs/html/designer/api-doc.html` 保持 100% 一致。
- 异常处理必须返回标准的 JSON 错误结构，禁止暴露原始报错堆栈。
- 所有数据库操作使用 pgxpool 连接池，禁止直接创建连接。
- 手机号使用 AES-256-GCM 加密存储，API 仅返回脱敏格式。
- 匹配引擎操作必须通过 Worker Pool 异步处理。
