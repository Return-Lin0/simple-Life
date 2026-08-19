# SimpleLife · 生活待办与记事本

一个面向日常生活的 Web 应用：待办事项、记事本、习惯打卡、纪念日倒计时，配备完整账号体系与实时提醒。

## 功能特性

- **待办事项**：新建 / 编辑 / 删除 / 完成恢复、时间与时间段、优先级、分类、标签、搜索筛选、日/周/月重复、逾期醒目提示、待办一键转记事
- **今日视图**：今天要做什么一目了然，逾期事项自动置顶
- **日历视图**：月历查看待办与纪念日，点击日期直接新建
- **记事本**：快速记录灵感、购物清单、日记片段
- **习惯打卡**：每天打卡、连续坚持天数、未来日期拦截、同日防重复
- **纪念日**：生日/纪念日倒计时（“还有 X 天 / 就是今天”）、每年重复、提前提醒
- **提醒系统**：到点通过 SSE 实时推送 + 浏览器通知 + 站内提醒中心
- **账号安全**：JWT 双 Token（Access + Refresh）、Refresh 轮换与重放吊销、bcrypt 密码哈希、数据库密码 AES-256-GCM 加密存储、登录失败锁定（可配置）

## 技术栈

| 端 | 技术 |
|----|------|
| 前端 | Vue 3 + TypeScript + Vite + Pinia + Vue Router + Element Plus + Day.js |
| 后端 | Go + Gin + GORM + JWT（golang-jwt/v5） |
| 数据 | MySQL 8（兼容 5.7）+ Redis 7 |
| 消息 | RabbitMQ 3（提醒异步投递、重试、死信） |
| 实时推送 | SSE（fetch 流式，自动重连） |

## 项目结构

```
simple-Life/
├── back_end/             # Go 后端（api / scheduler / worker 三个进程）
├── front_end/            # Vue 3 前端
├── 需求文档.md            # 需求定义
├── 技术设计文档.md         # 前后端技术方案
├── 测试用例文档.md         # 159 条测试用例
├── api_test.ps1          # 接口自动化测试（44 个用例）
├── docker-compose.yml    # RabbitMQ 容器编排
├── start.bat / start.ps1 # 一键启动
└── stop.bat / stop.ps1   # 一键停止
```

## 快速开始（推荐）

### 前置条件

- MySQL 已启动（默认账号 root / root，如 phpStudy 中的 MySQL 5.7）
- Redis 已启动（无密码，默认 6379）
- Docker 可用（用于启动 RabbitMQ，账号 admin / password）
- Go 1.22+ 与 Node.js 20+

### 一键启动

```powershell
# 在项目根目录 simple-Life 下执行
start.bat                          # 启动后端（双击即可）
start.bat -Frontend                # 同时启动前端

# 或使用 PowerShell 脚本
powershell -ExecutionPolicy Bypass -File start.ps1 -Frontend

# 停止
stop.bat
```

脚本会自动完成：启动 RabbitMQ 容器 → 生成会话级密钥并把数据库密码 `root` 加密为密文写入 `back_end/config/config.yaml` → 创建 `vibe` 数据库 → 编译后端 → 以隐藏窗口启动 api / scheduler / worker 三个进程。

启动完成后访问：

- 前端页面：http://localhost:5173
- 后端 API：http://localhost:8080
- RabbitMQ 管理面板：http://localhost:15672（admin / password）

> 说明：密钥只在启动脚本会话内有效，配置文件中仅保存密文，因此请统一使用 start/stop 脚本启停。若进程启动失败，查看 `back_end/logs/*.log`。

## 手动启动（可选）

```powershell
# 后端（三个终端，均在 back_end 目录）
go run ./cmd/api
go run ./cmd/scheduler
go run ./cmd/worker

# 前端（front_end 目录）
npm install
npm run dev
```

## 接口测试

按《测试用例文档.md》编写了 44 个自动化用例，覆盖认证、标签、待办、重复事项回归、记事、打卡、纪念日、提醒联动与送达、搜索、安全：

```powershell
# 需后端已启动（start.bat）
powershell -ExecutionPolicy Bypass -File api_test.ps1
```

## 配置说明

后端配置位于 `back_end/config/config.yaml`（由启动脚本自动生成，已加入 .gitignore）：

| 配置项 | 说明 |
|--------|------|
| `security.enable_login_lock` | 登录失败锁定（FR-20），默认关闭，改为 `true` 启用 |
| `security.api_rate_limit_per_minute` | 业务接口每用户每分钟限流 |
| `reminder.scan_interval_seconds` | 提醒调度扫描周期（默认 60 秒） |
| `cors.allowed_origins` | 前端域名白名单（默认 http://localhost:5173） |

密钥通过环境变量注入，不写入代码或配置：

| 环境变量 | 用途 |
|----------|------|
| `VIBE_DB_KEY` | AES-256-GCM 密钥（64 位十六进制），用于解密数据库密码 |
| `VIBE_JWT_SECRET` | JWT 签名密钥（≥ 32 字节） |

## 文档索引

- [需求文档.md](需求文档.md)：需求定义与账号/安全/技术决策
- [技术设计文档.md](技术设计文档.md)：数据库、接口、提醒链路、安全设计
- [测试用例文档.md](测试用例文档.md)：159 条验收用例与通过标准

## 常见问题

- **MySQL 未启动**：脚本会提示，请先在 phpStudy 面板启动 MySQL 5.7 再运行。
- **RabbitMQ 启动慢**：首次拉取镜像需要时间，脚本会等待 5672 端口就绪（最长 90 秒）。
- **后端进程启动失败**：查看 `back_end/logs/api.err.log`、`scheduler.err.log`、`worker.err.log`。
- **提醒不触发**：确认 scheduler 与 worker 均已启动，并保持页面打开接收 SSE 推送；浏览器通知需在首次提醒时允许授权。
