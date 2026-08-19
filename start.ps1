# VIBE 一键启动脚本
# 用法（在项目根目录 simple-Life 下执行）：
#   powershell -ExecutionPolicy Bypass -File start.ps1
#   powershell -ExecutionPolicy Bypass -File start.ps1 -Frontend   # 同时启动前端
#
# 前置条件：
#   - MySQL 已启动（root / root，phpStudy 中启动 MySQL 5.7，端口 3306）
#   - Redis 已启动（无密码，默认 6379）
#   - Docker 可用（用于运行 RabbitMQ，账号 admin / password）

param(
    [switch]$Frontend,  # 是否同时启动前端（npm run dev）
    [switch]$NoRabbit   # 跳过 RabbitMQ 启动（已手动启动时使用）
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServerDir = Join-Path $Root 'back_end'
$WebDir = Join-Path $Root 'front_end'
Set-Location $ServerDir

# 启动前先停止可能仍在运行的旧进程，避免二进制文件被占用
foreach ($name in 'api', 'scheduler', 'worker') {
    $pidFile = "pids\$name.pid"
    if (Test-Path $pidFile) {
        $procId = Get-Content $pidFile -ErrorAction SilentlyContinue
        if ($procId) {
            Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
            Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host ''
Write-Host '==============================================' -ForegroundColor Cyan
Write-Host '  VIBE 一键启动' -ForegroundColor Cyan
Write-Host '==============================================' -ForegroundColor Cyan

# ---------- 1. 检查 MySQL ----------
Write-Host ''
Write-Host '[1/5] 检查 MySQL...' -ForegroundColor Yellow
$mysqlUp = Get-NetTCPConnection -LocalPort 3306 -State Listen -ErrorAction SilentlyContinue
if (-not $mysqlUp) {
    Write-Host '  MySQL 未启动！请先在 phpStudy 面板中启动 MySQL 5.7，再重新运行本脚本。' -ForegroundColor Red
    exit 1
}
Write-Host '  MySQL 已就绪 ✓' -ForegroundColor Green

# ---------- 2. 启动 RabbitMQ（Docker） ----------
if (-not $NoRabbit) {
    Write-Host ''
    Write-Host '[2/5] 启动 RabbitMQ（Docker）...' -ForegroundColor Yellow
    docker compose -f (Join-Path $Root 'docker-compose.yml') up -d rabbitmq
    if ($LASTEXITCODE -ne 0) {
        Write-Host '  RabbitMQ 启动命令失败，请确认 Docker 已启动。' -ForegroundColor Red
        exit 1
    }
    # 等待 5672 端口就绪（最长 90 秒）
    $ready = $false
    for ($i = 0; $i -lt 18; $i++) {
        Start-Sleep -Seconds 5
        if (Test-NetConnection -ComputerName 127.0.0.1 -Port 5672 -InformationLevel Quiet) {
            $ready = $true
            break
        }
    }
    if (-not $ready) {
        Write-Host '  RabbitMQ 90 秒内未就绪，请检查 docker logs vibe-rabbitmq。' -ForegroundColor Red
        exit 1
    }
    Write-Host '  RabbitMQ 已就绪 ✓' -ForegroundColor Green
} else {
    Write-Host ''
    Write-Host '[2/5] 跳过 RabbitMQ 启动（-NoRabbit）' -ForegroundColor Yellow
}

# ---------- 3. 生成密钥 + 加密数据库密码 + 生成配置文件 ----------
Write-Host ''
Write-Host '[3/5] 生成密钥与配置文件...' -ForegroundColor Yellow

# 每次启动生成会话级密钥；配置中只保存密文，不落明文
$key = -join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
$env:VIBE_DB_KEY = $key
$env:VIBE_JWT_SECRET = ('vibe-jwt-' + $key)

$cipher = (go run ./cmd/encrypt -plaintext 'root' | Select-Object -Last 1).Trim()
if (-not $cipher) {
    Write-Host '  数据库密码加密失败，请确认 Go 环境正常。' -ForegroundColor Red
    exit 1
}

New-Item -ItemType Directory -Force -Path 'config', 'logs', 'pids', 'bin' | Out-Null

@"
server:
  port: 8080
  mode: debug
mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  dbname: vibe
  password_cipher: "$cipher"
  max_open_conns: 20
  max_idle_conns: 5
redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
rabbitmq:
  url: amqp://admin:password@127.0.0.1:5672/
jwt:
  secret_env: VIBE_JWT_SECRET
  access_ttl_minutes: 15
  refresh_ttl_days: 7
security:
  enable_login_lock: false
  max_login_failures: 5
  lock_minutes: 15
  api_rate_limit_per_minute: 120
cors:
  allowed_origins:
    - http://localhost:5173
logger:
  level: debug
reminder:
  scan_interval_seconds: 60
  window_seconds: 60
  compensation_batch: 50
"@ | Set-Content -Path 'config\config.yaml' -Encoding UTF8

# ---------- 4. 创建数据库并编译后端 ----------
Write-Host ''
Write-Host '[4/5] 创建数据库并编译后端...' -ForegroundColor Yellow
# PowerShell 5.1 会把 mysql 的 stderr 告警当成错误，这里临时放开 ErrorAction 并忽略输出
$oldEAP = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
mysql -uroot -proot -h127.0.0.1 -e "CREATE DATABASE IF NOT EXISTS vibe DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>&1 | Out-Null
$ErrorActionPreference = $oldEAP

go build -o 'bin\api.exe' ./cmd/api
go build -o 'bin\scheduler.exe' ./cmd/scheduler
go build -o 'bin\worker.exe' ./cmd/worker
if ($LASTEXITCODE -ne 0) {
    Write-Host '  后端编译失败，请检查上方错误信息。' -ForegroundColor Red
    exit 1
}
Write-Host '  数据库与编译就绪 ✓' -ForegroundColor Green

# ---------- 5. 启动三个后端进程 ----------
Write-Host ''
Write-Host '[5/5] 启动后端进程...' -ForegroundColor Yellow

$jobs = @(
    @{ Name = 'api';        Exe = 'bin\api.exe' }
    @{ Name = 'scheduler';  Exe = 'bin\scheduler.exe' }
    @{ Name = 'worker';     Exe = 'bin\worker.exe' }
)

Get-ChildItem 'pids\*.pid' -ErrorAction SilentlyContinue | Remove-Item -Force
foreach ($job in $jobs) {
    $p = Start-Process -FilePath $job.Exe `
        -WorkingDirectory $ServerDir `
        -WindowStyle Hidden `
        -RedirectStandardOutput (Join-Path $ServerDir "logs\$($job.Name).log") `
        -RedirectStandardError (Join-Path $ServerDir "logs\$($job.Name).err.log") `
        -PassThru
    $p.Id | Set-Content -Path "pids\$($job.Name).pid"
}

# 等待 3 秒后检查进程是否存活
Start-Sleep -Seconds 3
$alive = @()
foreach ($job in $jobs) {
    $procId = Get-Content "pids\$($job.Name).pid" -ErrorAction SilentlyContinue
    if ($procId -and (Get-Process -Id $procId -ErrorAction SilentlyContinue)) {
        $alive += $job.Name
        Write-Host "  $($job.Name) 已启动 ✓" -ForegroundColor Green
    } else {
        Write-Host "  $($job.Name) 启动失败 ✗ 请查看 logs\$($job.Name).err.log" -ForegroundColor Red
    }
}

if ($alive.Count -lt 3) {
    Write-Host ''
    Write-Host '部分进程启动失败，常见原因：' -ForegroundColor Yellow
    Write-Host '  - MySQL 未启动或账号密码不是 root/root'
    Write-Host '  - RabbitMQ 账号不是 admin / password'
    Write-Host '  - 详细日志：back_end\logs\*.log'
}

# ---------- 可选：启动前端 ----------
if ($Frontend) {
    Write-Host ''
    Write-Host '正在启动前端...' -ForegroundColor Yellow
    Start-Process -FilePath 'cmd.exe' -ArgumentList '/c', 'npm run dev' `
        -WorkingDirectory $WebDir `
        -WindowStyle Hidden `
        -RedirectStandardOutput (Join-Path $ServerDir 'logs\frontend.log') `
        -RedirectStandardError (Join-Path $ServerDir 'logs\frontend.err.log')
}

Write-Host ''
Write-Host '==============================================' -ForegroundColor Cyan
Write-Host '  启动完成！' -ForegroundColor Green
Write-Host '  后端 API       : http://localhost:8080'
Write-Host '  RabbitMQ 面板  : http://localhost:15672 (admin / password)'
if ($Frontend) {
    Write-Host '  前端页面       : http://localhost:5173'
} else {
    Write-Host '  前端（可选）   : 执行 start.ps1 -Frontend 或 npm run dev（在 front_end 目录）'
}
Write-Host '  停止服务       : stop.ps1'
Write-Host '==============================================' -ForegroundColor Cyan
Write-Host ''
