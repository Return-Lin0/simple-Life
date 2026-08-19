# VIBE 停止脚本：结束三个后端进程（RabbitMQ 容器保留，可用 -StopRabbit 一并停止）
param([switch]$StopRabbit)

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServerDir = Join-Path $Root 'back_end'
Set-Location $ServerDir

$names = 'api', 'scheduler', 'worker'
foreach ($name in $names) {
    $pidFile = "pids\$name.pid"
    if (Test-Path $pidFile) {
        $procId = Get-Content $pidFile -ErrorAction SilentlyContinue
        if ($procId) {
            Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
            Write-Host "已停止 $name (PID $procId)" -ForegroundColor Green
        }
        Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
    }
}

if ($StopRabbit) {
    docker compose -f (Join-Path $Root 'docker-compose.yml') stop rabbitmq
    Write-Host '已停止 RabbitMQ 容器（数据保留）' -ForegroundColor Green
}

Write-Host '停止完成' -ForegroundColor Cyan
