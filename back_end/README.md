# VIBE 后端服务

基于 Go + Gin + MySQL + Redis + RabbitMQ 的待办/记事/打卡/纪念日 Web 后端，实现详见《技术设计文档.md》，验收用例见《测试用例文档.md》。

## 技术栈

| 组件 | 用途 |
|------|------|
| Go 1.26 + Gin | HTTP API |
| MySQL 8 | 业务数据 |
| Redis 7 | Refresh Token 白名单/黑名单、限流、幂等去重、SSE 事件桥接 |
| RabbitMQ 3.12 | 提醒任务异步投递、重试、死信 |

## 目录结构

```
server/
├── cmd/
│   ├── api/          # HTTP API 进程（含 SSE 长连接）
│   ├── scheduler/    # 提醒调度器（扫描到期任务 → 发布消息）
│   ├── worker/       # 提醒消费者（幂等处理 → 推送 SSE）
│   └── encrypt/      # 数据库密码加密工具
├── config/           # 配置模板（config.yaml 不入库）
├── internal/
│   ├── auth/         # JWT、AES-256-GCM、bcrypt
│   ├── config/       # 配置加载与敏感项解密
│   ├── database/     # MySQL 连接与自动迁移
│   ├── model/        # GORM 模型（10 张表）
│   ├── repository/   # 数据访问层
│   ├── service/      # 业务层（事务编排）
│   ├── handler/      # HTTP 处理器
│   ├── middleware/   # 鉴权/限流/日志/CORS/恢复
│   ├── mq/           # RabbitMQ 拓扑与收发
│   ├── notify/       # SSE 连接中枢
│   └── router/       # 路由注册
```

## 快速开始

### 1. 准备中间件

本地安装并启动 MySQL 8、Redis 7、RabbitMQ 3.12（或使用 Docker）。

```sql
CREATE DATABASE vibe DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'vibe'@'localhost' IDENTIFIED BY '你的数据库密码';
GRANT ALL PRIVILEGES ON vibe.* TO 'vibe'@'localhost';
FLUSH PRIVILEGES;
```

### 2. 准备密钥与配置

生成 AES 密钥（64 位十六进制）与 JWT 密钥，并设置环境变量：

```powershell
# 生成 32 字节随机密钥的十六进制表示
$key = -join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
$env:VIBE_DB_KEY = $key
$env:VIBE_JWT_SECRET = ('vibe-jwt-' + $key)
```

加密数据库密码，输出写入配置：

```powershell
go run ./cmd/encrypt -plaintext '你的数据库密码'
```

复制配置模板并填写（数据库密码只填密文）：

```powershell
Copy-Item config\config.example.yaml config\config.yaml
```

> 安全说明：`VIBE_DB_KEY` 与 `VIBE_JWT_SECRET` 只通过环境变量注入；
> `config.yaml` 已被 `.gitignore` 排除，禁止提交明文密码。

### 3. 启动三个进程

```powershell
go run ./cmd/api        # HTTP API :8080
go run ./cmd/scheduler  # 提醒调度
go run ./cmd/worker     # 提醒消费
```

建表由 `database.NewMySQL` 的 AutoMigrate 自动完成；生产环境建议替换为显式 SQL 迁移脚本。

### 4. 验证

```powershell
curl http://localhost:8080/api/v1/auth/register `
  -H "Content-Type: application/json" `
  -d '{"username":"demo","password":"Abc12345","nickname":"演示"}'
```

## 配置说明

| 配置项 | 说明 |
|--------|------|
| `security.enable_login_lock` | FR-20 登录失败锁定，默认关闭，确认启用后改为 `true` |
| `security.api_rate_limit_per_minute` | 业务接口每用户每分钟限流 |
| `reminder.scan_interval_seconds` | 调度扫描周期（默认 60 秒） |
| `cors.allowed_origins` | 前端域名白名单（开发默认 `http://localhost:5173`） |

## 提醒链路

1. 创建待办/纪念日时，在同一事务内写入 `reminder_tasks`；
2. 调度器每分钟扫描到期任务 → 发布到 RabbitMQ `vibe.reminder`；
3. 消费者幂等处理（Redis `SETNX`）→ 写入提醒中心 → Redis Pub/Sub 通知 API 进程；
4. API 进程通过 SSE（`GET /api/v1/events`）实时推送到浏览器；
5. 失败消息自动重试 3 次，超限进入死信队列，调度器对 `fail_count < 5` 的任务补偿重发。
