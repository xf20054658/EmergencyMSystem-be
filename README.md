# 🆘 EmergencyMSystem 应急协同平台 — 后端

> **广西洪涝灾害应急管理多端协同平台**  
> 集中指挥 + P2P 混合架构 · 微信小程序 + PC 指挥中心

---

## 🤝 献给灾区

**本项目无条件捐献给受灾地区及各级应急管理部门使用。**

- ✅ 完全开源，**无偿使用**
- ✅ 提供**免费技术部署支持**
- ✅ 可定制化适配各灾害场景（洪涝/地震/台风）
- ✅ 数据安全：手机号 AES-256-GCM 加密，AXB 虚拟号码隐私通话

> 如有部署、定制或技术咨询需求，欢迎提交 Issue 或联系项目维护者。  
> **灾情不等人，技术无边界。**

---

## 📋 项目概览

一个面向洪涝灾害场景的应急协同系统，采用**集中指挥 + P2P 去中心化辅助**的混合拓扑：

- **市民** → 微信小程序发起求助 / SOS 一键报警
- **志愿者** → P2P 就近接单，技能匹配，实时定位共享
- **指挥中心** → PC 端全量监控，集中调度，应急物资管理

### 核心指标

| 指标 | 设计值 |
|-----|--------|
| 单机 QPS | 1000+（Worker Pool + pgxpool 连接池） |
| 匹配延迟 | < 5 秒（六维匹配算法，20 个 Worker） |
| 数据加密 | AES-256-GCM 手机号加密存储 |
| WebSocket 实时推送 | 新求助/匹配变更/GPS轨迹/告警 |
| 限流策略 | 令牌桶 300 RPM，按 ClientIP/UserID |
| 评分 | 请根据自己的实际需求打分，也可以聊聊**

---

## 🏗 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                     Mobile (微信小程序)                    │
│  求助上报 · SOS一键报警 · 志愿者接单 · 实时定位 · 通知    │
└──────────────────┬──────────────────────────────┬────────┘
                   │ WebSocket + REST API          │
                   ▼                              ▼
┌──────────────────────────────────────────────────────────┐
│                   EmergencyMSystem Backend                │
│                                                          │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────┐      │
│  │ Auth     │  │ Help Req  │  │ Match Engine     │      │
│  │ 微信登录 │  │ CRUD+SOS  │  │ Worker Pool(20)  │      │
│  │ JWT鉴权  │  │ 代报/特殊 │  │ 六维匹配算法     │      │
│  └──────────┘  └───────────┘  └──────────────────┘      │
│                                                          │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────┐      │
│  │ Provider │  │ Notify    │  │ Dashboard        │      │
│  │ 志愿者   │  │ 微信推送  │  │ KPI/图表/大屏    │      │
│  │ Tier管理  │  │ 偏好订阅  │  │ 物资/安置点/队伍 │      │
│  └──────────┘  └───────────┘  └──────────────────┘      │
│                                                          │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────┐      │
│  │ WebSocket│  │ GPS追踪   │  │ Communication    │      │
│  │ 实时推送  │  │ 轨迹共享  │  │ AXB隐私号       │      │
│  └──────────┘  └───────────┘  └──────────────────┘      │
└──────────────────────────────────────────────────────────┘
                   │
         ┌─────────┴────────────┐
         ▼                      ▼
    ┌──────────┐          ┌──────────┐
    │ PostgreSQL│         │  Redis   │
    │ 33 tables │         │  缓存    │
    │ GIN索引   │         │  会话    │
    │ 视图      │         │  队列    │
    └──────────┘          └──────────┘
```

---

## 🚀 快速开始

### 前置条件

- Go 1.21+
- PostgreSQL 14+
- Redis 7+

### 数据库初始化

```bash
# 完整 DDL 详见项目文档
psql -U postgres -d emergency_msystem -f docs/database/schema.sql

# 导入 Mock 数据（可选）
psql -U postgres -d emergency_msystem -f docs/database/mock-data.sql
```

### 启动服务

```bash
# 1. 配置环境变量
cp config/.env.example .env
# 编辑 .env，填写数据库连接等配置

# 2. 安装依赖
go mod tidy

# 3. 启动开发服务
go run main.go

# 4. 生产构建
go build -o bin/server main.go
./bin/server
```

### 验证

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"emergency-msystem"}
```

---

## 📡 API 一览

### 认证 `POST /api/v1/auth/login`

```json
// Request
{"code": "微信登录code"}
// Response
{"code": 0, "data": {"token": "jwt...", "is_new_user": true, "user": {...}}}
```

### 核心接口速览

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| **认证** | | | |
| POST | `/api/v1/auth/login` | 否 | 微信登录 |
| POST | `/api/v1/auth/login/phone` | 否 | 手机号验证码登录 |
| GET | `/api/v1/user/profile` | 是 | 用户信息 |
| **求助** | | | |
| POST | `/api/v1/help-requests` | 是 | 创建求助（含代报/特殊群体） |
| GET | `/api/v1/help-requests` | 是 | 列表（分页+筛选） |
| GET | `/api/v1/help-requests/:id` | 是 | 详情（含需求+匹配） |
| **SOS** | | | |
| POST | `/api/v1/sos/trigger` | 是 | SOS 触发（GPS/last_known兜底） |
| PUT | `/api/v1/sos/:id/confirm` | 是 | 确认/取消/误报 |
| **匹配** | | | |
| PUT | `/api/v1/matches/:id/accept` | 志愿者 | 接受匹配 |
| PUT | `/api/v1/matches/:id/waiting` | 志愿者 | 设置等待（道路中断） |
| POST | `/api/v1/matches/:id/reinforce` | 志愿者 | 请求增援 |
| **志愿者** | | | |
| POST | `/api/v1/providers/register` | 是 | 注册志愿者 |
| POST | `/api/v1/providers/gps` | 志愿者 | 实时 GPS 上报 |
| **通知** | | | |
| POST | `/api/v1/notifications/subscribe` | 是 | 订阅微信通知 |
| GET | `/api/v1/notifications/preferences` | 是 | 通知偏好 |
| **指挥中心** | | | |
| GET | `/api/v1/dashboard/kpi` | 是 | KPI 概览 |
| GET | `/api/v1/dashboard/match-stats` | 是 | 匹配效率统计 |
| **WebSocket** | | | |
| WS | `/ws/command` | 是 | 指挥中心实时通道 |
| WS | `/ws/user/:id` | 是 | 用户端实时通道 |

**完整 API 文档** → `docs/html/designer/api-doc.html`

---

## 🧠 核心设计

### 六维匹配算法

```
得分 = 技能匹配(30%) × 距离得分(25%) × Tier分(20%)
     × 密度因子(10%) × 优先级(10%) × 防滥用(5%)
```

### 匹配状态机

```
pending → accepted → enroute → waiting(可逆) → completed
  ↓         ↓           ↓
rejected  timeout    unreachable(30min GPS未移动)
  ↓                   ↓
escalated ←──────────┘
```

### 混合路由

| 紧急度 | 路由 | 说明 |
|-------|------|------|
| `critical`（危急） | → **集中（Centralized）** | 指挥中心直接派单 |
| `high` / `medium` / `low` | → **P2P** | 志愿者接单，超时升级 |

### 数据安全

- 手机号 → AES-256-GCM 加密存储（`phone_encrypted`）
- API 仅返回脱敏格式（`138****6288`）
- 志愿者与求助者通过 AXB 虚拟号码通话，真实号码互不可见
- JWT 无状态鉴权，Token 7 天有效期

---

## 🛠 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| **语言** | Go 1.21+ | 高并发、低内存 |
| **Web 框架** | Gin v1.9+ | 高性能 HTTP 路由器 |
| **数据库** | PostgreSQL 14+ | pgxpool 连接池（50 连接） |
| **缓存** | Redis 7+ | 缓存 + 会话管理 |
| **WebSocket** | Gorilla WebSocket | 实时双向通信 |
| **鉴权** | golang-jwt v5 | 无状态 JWT |
| **加密** | crypto/aes | AES-256-GCM |
| **限流** | 自研令牌桶 | 无外部依赖 |

---

## 📁 项目结构

```
src/backend/
├── main.go                        # 入口：服务启动 + 优雅退出
├── config/                        # 配置管理
├── internal/
│   ├── database/                  # PostgreSQL pgxpool
│   ├── redis/                     # Redis 连接
│   ├── models/                    # 33 个领域模型
│   ├── dto/                       # 10 个请求/响应 DTO + 单元测试
│   ├── handler/                   # 8 个 HTTP Handler
│   ├── service/                   # 认证 + 匹配引擎（Worker Pool）
│   ├── middleware/                 # JWT / 日志 / 令牌桶限流
│   ├── websocket/                 # WebSocket Hub
│   └── router/                    # 36+ 路由注册
├── pkg/
│   ├── jwt/                       # JWT 生成/解析 + 单元测试
│   ├── crypto/                    # AES加密/脱敏/哈希 + 单元测试
│   └── response/                  # 统一响应格式 + 单元测试
├── go.mod / go.sum
└── BACKEND_GUIDE.md               # 完整开发指南
```

---

## ✅ 测试

```bash
# 后端单元测试
go test ./pkg/... ./internal/dto/...

# 带覆盖率报告
go test ./pkg/... ./internal/dto/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## 📄 许可

本项目 **无条件捐献给灾害应急领域**。任何组织和个人均可自由使用、修改和分发。

- 用于灾害应急救援的：✅ **无偿使用**
- 用于商业目的：✅ **免费，欢迎联系开发者**
- 用于学习研究：✅ **欢迎 Fork 和 PR**

**灾情不等人，技术无边界。** 🌊

---

---

## 🙋 加入我们

**用技术为社会贡献一份力量。**

本项目欢迎所有技术人员参与维护：

- 🛠️ **代码贡献** — Fork → 改代码 → PR，包括但不限于 Bug 修复、性能优化、新功能
- 💡 **产品建议** — 提交 Issue 讨论设计改进、场景扩展
- 🌐 **生态建设** — 接入更多灾情数据源（气象/水文/地质），适配更多终端
- 📖 **文档完善** — 案例文档、部署教程、运维手册

> 没有小贡献，只有不行动。一个人的代码，可能救一座城。

## 📢 我们需要您推荐

- 🏛️ **有政府/应急管理部门资源的朋****请将本项目推荐给相关负责人员**
  - 应急管理厅/局、水利局、气象局
  - 消防救援队伍、红十字会、民间救援组织
  - 街道办事处、乡镇政府等基层单位
- 🏪 **有企业资源的朋****可协助对接通信运营商（AXB 隐私号）、地图服务（高德/百度）、短信通道等**
- 📱 **微信小程序认证加速**
  - 本项目的微信小程序**急需完成主体认证和服务类目审核**
  - 如有微信官方渠道或认证加速资源，请与我们联系
  - 其他合规认证如：ICP 备案、等保测评、数据安全评估等也在推进中

**您的每一次转发和推荐，都在为受灾群众争取救援时间。**

---

> **项目维护者：** 心程  
> **仓库地址：** https://github.com/xf20054658/EmergencyMSystem-be  
> **如有应急部署需求，请直接提交 Issue，我们会在 24 小时内响应。  
> **联系方式：** 通过 GitHub Issue 或飞书联系**
