# SimpleLife 后端接口自动化测试脚本
# 依据《测试用例文档.md》覆盖：认证、待办、标签、记事、打卡、纪念日、提醒、搜索、安全。
# 运行：powershell -ExecutionPolicy Bypass -File api_test.ps1

$ErrorActionPreference = 'Stop'
$Base = 'http://localhost:8080/api/v1'
$RunId = Get-Date -Format 'yyyyMMddHHmmss'
$env:MYSQL_PWD = 'root'   # 避免 mysql 命令行密码告警干扰
$Results = [System.Collections.Generic.List[object]]::new()

# ---------- 工具函数 ----------
function Invoke-Api {
    param(
        [string]$Method,
        [string]$Path,
        [string]$Token = '',
        $Body = $null
    )
    $tmpOut = Join-Path $env:TEMP ("slapi_" + [guid]::NewGuid().ToString('N') + ".json")
    $args = @('-s', '-o', $tmpOut, '-w', '%{http_code}', '-X', $Method, "$Base$Path")
    if ($Token) { $args += @('-H', "Authorization: Bearer $Token") }
    if ($null -ne $Body) {
        $tmpBody = Join-Path $env:TEMP ("slbody_" + [guid]::NewGuid().ToString('N') + ".json")
        $json = $Body | ConvertTo-Json -Depth 6 -Compress
        # PowerShell 5.1 的 Set-Content -Encoding UTF8 会写入 BOM，导致 JSON 解析失败，
        # 因此使用无 BOM 的 UTF8Encoding 直接写文件。
        [System.IO.File]::WriteAllText($tmpBody, $json, (New-Object System.Text.UTF8Encoding($false)))
        $args += @('-H', 'Content-Type: application/json', '--data-binary', "@$tmpBody")
    }
    $code = (& curl.exe @args 2>$null)
    $raw = ''
    if (Test-Path $tmpOut) {
        # 必须按 UTF-8 读取：默认 ANSI(GBK) 会把中文尾字节与 JSON 引号拼成双字节字符，
        # 导致 JSON 结构损坏、ConvertFrom-Json 失败。
        $raw = Get-Content $tmpOut -Raw -Encoding UTF8 -ErrorAction SilentlyContinue
        Remove-Item $tmpOut -Force -ErrorAction SilentlyContinue
    }
    if ($tmpBody -and (Test-Path $tmpBody)) { Remove-Item $tmpBody -Force -ErrorAction SilentlyContinue }
    $parsed = $null
    if ($raw) {
        try { $parsed = $raw | ConvertFrom-Json } catch { }
    }
    return @{ Status = [int]$code; Data = $parsed; Raw = $raw }
}

function Assert-Status {
    param($Resp, [int]$Expected, [string]$Name)
    if ($Resp.Status -ne $Expected) {
        throw "[$Name] 期望 HTTP $Expected，实际 $($Resp.Status)：$($Resp.Raw)"
    }
}

function Assert-True {
    param([bool]$Cond, [string]$Name)
    if (-not $Cond) { throw "[$Name] 断言失败" }
}

function Sql-Query {
    param([string]$Query)
    $out = mysql --host=127.0.0.1 -uroot --database=vibe -N -e $Query 2>$null | Out-String
    return $out.Trim()
}

function Test-Case {
    param([string]$Name, [scriptblock]$Body)
    try {
        $null = & $Body
        $Results.Add([pscustomobject]@{ Name = $Name; Result = 'PASS' })
        Write-Host "  [PASS] $Name" -ForegroundColor Green
    }
    catch {
        $msg = $_.Exception.Message
        $Results.Add([pscustomobject]@{ Name = $Name; Result = "FAIL: $msg" })
        Write-Host "  [FAIL] $Name -> $msg" -ForegroundColor Red
    }
}

function New-TestUser {
    $u = "t_${RunId}_" + (Get-Random -Max 99999)
    $r = Invoke-Api POST '/auth/register' -Body @{ username = $u; password = 'Abc12345'; nickname = '测试用户' }
    if ($r.Status -ne 200) { throw "测试用户注册失败: $($r.Raw)" }
    $l = Invoke-Api POST '/auth/login' -Body @{ username = $u; password = 'Abc12345' }
    return @{ User = $u; Token = $l.Data.data.access_token }
}

Write-Host ''
Write-Host '================ SimpleLife 接口测试 ================' -ForegroundColor Cyan

# ---------- 1. 认证 ----------
Write-Host ''
Write-Host '[认证]' -ForegroundColor Yellow
$u1 = "t_${RunId}_auth1"
Test-Case '注册成功' {
    $r = Invoke-Api POST '/auth/register' -Body @{ username = $u1; password = 'Abc12345'; nickname = '甲' }
    Assert-Status $r 200 '注册成功'
}
Test-Case '重复用户名注册被拒(409)' {
    $r = Invoke-Api POST '/auth/register' -Body @{ username = $u1; password = 'Abc12345'; nickname = '甲' }
    Assert-Status $r 409 '重复用户名'
}
Test-Case '弱密码被拒(400)' {
    $r = Invoke-Api POST '/auth/register' -Body @{ username = "t_${RunId}_weak"; password = '12345678'; nickname = '乙' }
    Assert-Status $r 400 '弱密码'
}
Test-Case '登录成功返回双Token' {
    $r = Invoke-Api POST '/auth/login' -Body @{ username = $u1; password = 'Abc12345' }
    Assert-Status $r 200 '登录'
    if ($null -eq $r.Data.data.access_token) { throw "缺少 access_token，原始响应：$($r.Raw)" }
}
Test-Case '错误密码与不存在用户提示一致' {
    $r1 = Invoke-Api POST '/auth/login' -Body @{ username = $u1; password = 'Wrong12345' }
    $r2 = Invoke-Api POST '/auth/login' -Body @{ username = "t_${RunId}_nobody"; password = 'Wrong12345' }
    Assert-Status $r1 401 '错误密码'
    Assert-Status $r2 401 '不存在用户'
    Assert-True ($r1.Data.message -eq $r2.Data.message) '提示不一致（用户枚举风险）'
}
Test-Case '无Token访问受保护接口(401)' {
    $r = Invoke-Api GET '/auth/me'
    Assert-Status $r 401 '未授权'
}

$A = New-TestUser
Test-Case '带Token获取用户信息且不泄露哈希' {
    $r = Invoke-Api GET '/auth/me' -Token $A.Token
    Assert-Status $r 200 '用户信息'
    Assert-True ($r.Data.data.username -eq $A.User) '用户名不匹配'
    Assert-True (-not ($r.Raw -match 'password_hash')) '响应泄露密码哈希'
}
Test-Case '篡改Token被拒(401)' {
    $bad = $A.Token.Substring(0, $A.Token.Length - 4) + 'AAAA'
    $r = Invoke-Api GET '/auth/me' -Token $bad
    Assert-Status $r 401 '篡改Token'
}
Test-Case '登出后Token失效(401)' {
    $B = New-TestUser
    $null = Invoke-Api POST '/auth/logout' -Token $B.Token
    $r = Invoke-Api GET '/auth/me' -Token $B.Token
    Assert-Status $r 401 '登出后Token'
}
Test-Case '修改昵称成功' {
    $r = Invoke-Api PUT '/auth/profile' -Token $A.Token -Body @{ nickname = '新昵称' }
    Assert-Status $r 200 '修改昵称'
    Assert-True ($r.Data.data.nickname -eq '新昵称') '昵称未更新'
}
Test-Case '修改昵称空值被拒(400)' {
    $r = Invoke-Api PUT '/auth/profile' -Token $A.Token -Body @{ nickname = '  ' }
    Assert-Status $r 400 '空昵称'
}
Test-Case '原密码错误修改被拒(400)' {
    $PW = New-TestUser
    $r = Invoke-Api POST '/auth/change-password' -Token $PW.Token -Body @{ old_password = 'Wrong12345'; new_password = 'NewPass123' }
    Assert-Status $r 400 '原密码错误'
}
Test-Case '新密码强度不足被拒(400)' {
    $PW = New-TestUser
    $r = Invoke-Api POST '/auth/change-password' -Token $PW.Token -Body @{ old_password = 'Abc12345'; new_password = '12345678' }
    Assert-Status $r 400 '弱新密码'
}
Test-Case '修改密码成功后旧密码失效新密码可登录' {
    $PW = New-TestUser
    $r = Invoke-Api POST '/auth/change-password' -Token $PW.Token -Body @{ old_password = 'Abc12345'; new_password = 'NewPass123' }
    Assert-Status $r 200 '修改密码'
    $rOld = Invoke-Api POST '/auth/login' -Body @{ username = $PW.User; password = 'Abc12345' }
    Assert-Status $rOld 401 '旧密码登录'
    $rNew = Invoke-Api POST '/auth/login' -Body @{ username = $PW.User; password = 'NewPass123' }
    Assert-Status $rNew 200 '新密码登录'
    $rOldTok = Invoke-Api GET '/auth/me' -Token $PW.Token
    Assert-Status $rOldTok 401 '旧会话吊销'
}

# ---------- 2. 标签 ----------
Write-Host ''
Write-Host '[标签]' -ForegroundColor Yellow
$TagId = $null
Test-Case '创建标签' {
    $r = Invoke-Api POST '/tags' -Token $A.Token -Body @{ name = '工作'; color = '#6c7bff' }
    Assert-Status $r 200 '创建标签'
    $script:TagId = $r.Data.data.id
}
Test-Case '重复标签名被拒(409)' {
    $r = Invoke-Api POST '/tags' -Token $A.Token -Body @{ name = '工作' }
    Assert-Status $r 409 '重复标签'
}
Test-Case '标签列表包含新标签' {
    $r = Invoke-Api GET '/tags' -Token $A.Token
    Assert-Status $r 200 '标签列表'
    $hit = @($r.Data.data | Where-Object { $_.id -eq $TagId })
    if ($hit.Count -ne 1) { throw "列表缺少标签 id=$TagId，原始响应：$($r.Raw)" }
}
Test-Case '编辑并删除标签' {
    $r1 = Invoke-Api PUT "/tags/$TagId" -Token $A.Token -Body @{ name = '职场'; color = '#5a69e6' }
    Assert-Status $r1 200 '编辑标签'
    $r2 = Invoke-Api DELETE "/tags/$TagId" -Token $A.Token
    Assert-Status $r2 200 '删除标签'
}

# ---------- 3. 待办 ----------
Write-Host ''
Write-Host '[待办]' -ForegroundColor Yellow
$Todo1 = $null
Test-Case '最小信息新建待办' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = '买牛奶'; event_date = '2026-08-20' }
    Assert-Status $r 200 '最小新建'
    $script:Todo1 = $r.Data.data.id
}
Test-Case '完整信息新建（时间段+提醒+标签）' {
    $tag = (Invoke-Api POST '/tags' -Token $A.Token -Body @{ name = '重要' }).Data.data.id
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title                 = '14:00 与团队开会'
        description           = '准备周报'
        event_date            = '2026-08-21'
        start_time            = '14:00:00'
        end_time              = '15:30:00'
        priority              = 0
        category              = 'work'
        tags                  = @($tag)
        reminder_enabled      = $true
        remind_offset_minutes = 10
    }
    Assert-Status $r 200 '完整新建'
}
Test-Case '缺标题被拒(400)' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = ''; event_date = '2026-08-20' }
    Assert-Status $r 400 '空标题'
}
Test-Case '结束时间不晚于开始时间被拒(400)' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '时间非法'; event_date = '2026-08-20'
        start_time = '15:30:00'; end_time = '14:00:00'
    }
    Assert-Status $r 400 '非法时间段'
}
Test-Case '非法优先级/分类被拒(400)' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '枚举非法'; event_date = '2026-08-20'; priority = 9; category = 'unknown'
    }
    Assert-Status $r 400 '非法枚举'
}
Test-Case '引用他人标签被拒(404)' {
    $B = New-TestUser
    $tagB = (Invoke-Api POST '/tags' -Token $B.Token -Body @{ name = 'B的标签' }).Data.data.id
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '越权标签'; event_date = '2026-08-20'; tags = @($tagB)
    }
    Assert-Status $r 404 '他人标签'
}
Test-Case '今日视图只含今日与逾期' {
    $r = Invoke-Api GET '/todos?view=today' -Token $A.Token
    Assert-Status $r 200 '今日视图'
    foreach ($t in $r.Data.data) {
        if ($t.event_date -gt '2026-08-19') { throw "今日视图混入未来事项 $($t.event_date)" }
    }
}
Test-Case '完成/恢复/筛选/排序' {
    $r1 = Invoke-Api PATCH "/todos/$Todo1/status" -Token $A.Token -Body @{ status = 1 }
    Assert-Status $r1 200 '标记完成'
    $done = Invoke-Api GET '/todos?status=1' -Token $A.Token
    $hit = @($done.Data.data.list | Where-Object { $_.id -eq $Todo1 })
    if ($hit.Count -ne 1) { throw "已完成筛选缺失 id=$Todo1，原始响应：$($done.Raw)" }
    $r2 = Invoke-Api PATCH "/todos/$Todo1/status" -Token $A.Token -Body @{ status = 0 }
    Assert-Status $r2 200 '恢复未完成'
    $detail = Invoke-Api GET "/todos/$Todo1" -Token $A.Token
    Assert-True ($detail.Data.data.status -eq 0) '恢复后状态错误'
    $kw = Invoke-Api GET '/todos?keyword=%E5%BC%80%E4%BC%9A' -Token $A.Token
    Assert-True ($kw.Data.data.total -ge 1) '关键词搜索无结果'
    $work = Invoke-Api GET '/todos?category=work&sort_by=priority&order=asc' -Token $A.Token
    Assert-True ($work.Data.data.total -ge 1) '分类筛选无结果'
}
Test-Case '分页边界(page_size<=100)' {
    $r = Invoke-Api GET '/todos?page=1&page_size=999' -Token $A.Token
    Assert-Status $r 200 '分页'
    Assert-True ($r.Data.data.page_size -le 100) 'page_size 未限制'
}
Test-Case '删除待办并清理提醒任务' {
    $null = Invoke-Api DELETE "/todos/$Todo1" -Token $A.Token
    $tasks = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=1 AND target_id=$Todo1"
    Assert-True ($tasks -eq '0') '删除后提醒任务未清理'
}
Test-Case '越权访问他人待办(404)' {
    $B = New-TestUser
    $mine = (Invoke-Api POST '/todos' -Token $B.Token -Body @{ title = 'B的待办'; event_date = '2026-08-20' }).Data.data.id
    $r = Invoke-Api GET "/todos/$mine" -Token $A.Token
    Assert-Status $r 404 '越权待办'
}
Test-Case '待办转记事成功且防重复(409)' {
    $t = (Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = '转记事素材'; event_date = '2026-08-20' }).Data.data.id
    $r1 = Invoke-Api POST "/todos/$t/convert-to-note" -Token $A.Token
    Assert-Status $r1 200 '转记事'
    $r2 = Invoke-Api POST "/todos/$t/convert-to-note" -Token $A.Token
    Assert-Status $r2 409 '重复转换'
}
Test-Case '逾期事项标记' {
    # 过去日期现在不允许通过接口创建，改用 SQL 直接插入逾期数据验证标记逻辑
    $uid = (Invoke-Api GET '/auth/me' -Token $A.Token).Data.data.id
    $null = Sql-Query "INSERT INTO todos (user_id,title,event_date,status,created_at,updated_at,priority,category,recurrence_type,recurrence_rule,is_all_day,reminder_enabled) VALUES ($uid,'昨日未完成','2026-08-18',0,NOW(),NOW(),1,'other',0,'null',0,0)"
    $t = Sql-Query "SELECT id FROM todos WHERE user_id=$uid AND title='昨日未完成' ORDER BY id DESC LIMIT 1"
    $r = Invoke-Api GET "/todos/$t" -Token $A.Token
    if ($r.Data.data.overdue -ne $true) { throw "逾期标记缺失，原始响应：$($r.Raw)" }
}
Test-Case '创建过去日期被拒(400)' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = '过去日期'; event_date = '2026-08-18' }
    Assert-Status $r 400 '过去日期创建'
}
Test-Case '创建今天已过时间被拒(400)' {
    $nowT = Get-Date
    $pastT = $nowT.AddMinutes(-10)
    $pastClock = if ($pastT.Date -eq $nowT.Date) { $pastT.ToString('HH:mm:ss') } else { '00:00:00' }
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '已过时间'; event_date = $nowT.ToString('yyyy-MM-dd'); start_time = $pastClock
    }
    Assert-Status $r 400 '已过开始时间'
}
Test-Case '编辑逾期事项：保留原日期允许、改为新过去日期被拒' {
    $uid = (Invoke-Api GET '/auth/me' -Token $A.Token).Data.data.id
    $null = Sql-Query "INSERT INTO todos (user_id,title,event_date,status,created_at,updated_at,priority,category,recurrence_type,recurrence_rule,is_all_day,reminder_enabled) VALUES ($uid,'逾期可编辑','2026-08-18',0,NOW(),NOW(),1,'other',0,'null',0,0)"
    $id = Sql-Query "SELECT id FROM todos WHERE user_id=$uid AND title='逾期可编辑' ORDER BY id DESC LIMIT 1"
    $r1 = Invoke-Api PUT "/todos/$id" -Token $A.Token -Body @{ title = '逾期可编辑2'; event_date = '2026-08-18' }
    Assert-Status $r1 200 '保留原日期编辑'
    $r2 = Invoke-Api PUT "/todos/$id" -Token $A.Token -Body @{ title = '逾期可编辑3'; event_date = '2026-08-17' }
    Assert-Status $r2 400 '改为新过去日期'
}
Test-Case '提醒偏移超上限被拒(400)' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '偏移过大'; event_date = '2026-08-20'; start_time = '10:00:00'
        reminder_enabled = $true; remind_offset_minutes = 10000
    }
    Assert-Status $r 400 '提醒偏移超限'
}
Test-Case '每周重复缺少星期规则被拒(400)' {
    $r = Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '缺星期'; event_date = '2026-08-20'
        recurrence_type = 2; recurrence_rule = '{"weekdays":[]}'
    }
    Assert-Status $r 400 '每周缺星期'
}

# ---------- 4. 待办批量操作 ----------
Write-Host ''
Write-Host '[待办批量操作]' -ForegroundColor Yellow
Test-Case '批量标记完成与恢复' {
    $ids = @()
    1..3 | ForEach-Object {
        $ids += (Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = "批量待办$_"; event_date = '2026-08-20' }).Data.data.id
    }
    $r1 = Invoke-Api PATCH '/todos/batch-status' -Token $A.Token -Body @{ ids = @($ids[0], $ids[1]); status = 1 }
    Assert-Status $r1 200 '批量完成'
    Assert-True ($r1.Data.data.affected -eq 2) "批量完成影响数=$($r1.Data.data.affected)"
    $r2 = Invoke-Api PATCH '/todos/batch-status' -Token $A.Token -Body @{ ids = @($ids[0]); status = 0 }
    Assert-Status $r2 200 '批量恢复'
    Assert-True ($r2.Data.data.affected -eq 1) "批量恢复影响数=$($r2.Data.data.affected)"
}
Test-Case '批量删除并清理提醒任务' {
    $ids = @()
    1..3 | ForEach-Object {
        $ids += (Invoke-Api POST '/todos' -Token $A.Token -Body @{
            title = "批量删除$_"; event_date = '2026-08-21'; start_time = '14:00:00'
            reminder_enabled = $true; remind_offset_minutes = 5
        }).Data.data.id
    }
    $before = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=1 AND target_id IN ($($ids -join ','))"
    Assert-True ($before -eq '3') "删除前提醒任务=$before，应为3"
    $r = Invoke-Api POST '/todos/batch-delete' -Token $A.Token -Body @{ ids = $ids }
    Assert-Status $r 200 '批量删除'
    Assert-True ($r.Data.data.affected -eq 3) "批量删除影响数=$($r.Data.data.affected)"
    $after = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=1 AND target_id IN ($($ids -join ','))"
    Assert-True ($after -eq '0') "删除后提醒任务=$after，应清理为0"
}
Test-Case '批量参数校验（空数组被拒）' {
    $r = Invoke-Api PATCH '/todos/batch-status' -Token $A.Token -Body @{ ids = @(); status = 1 }
    Assert-Status $r 400 '空数组批量操作'
}

# ---------- 4. 重复事项（回归：时间重复添加修复验证） ----------
Write-Host ''
Write-Host '[重复事项回归]' -ForegroundColor Yellow
Test-Case '系列开始前不生成实例' {
    $root = (Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '每周例会'; event_date = '2026-08-20'; start_time = '14:00:00'
        recurrence_type = 2; recurrence_rule = '{"weekdays":[1,3,5]}'
    }).Data.data.id
    $null = Invoke-Api GET '/todos/calendar?start_date=2026-08-15&end_date=2026-08-31' -Token $A.Token
    $rows = Sql-Query "SELECT COUNT(*) FROM todos WHERE parent_id=$root AND event_date < '2026-08-20'"
    Assert-True ($rows -eq '0') "系列开始前存在实例 $rows 条"
}
Test-Case '重复调用日历接口不产生重复实例' {
    $root = (Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '防重复系列'; event_date = '2026-08-20'; start_time = '09:00:00'
        recurrence_type = 2; recurrence_rule = '{"weekdays":[1,3,5]}'
    }).Data.data.id
    1..3 | ForEach-Object {
        $null = Invoke-Api GET '/todos/calendar?start_date=2026-08-15&end_date=2026-08-31' -Token $A.Token
    }
    $rows = Sql-Query "SELECT CONCAT(COUNT(*),'-',COUNT(DISTINCT CONCAT(parent_id,'-',event_date))) FROM todos WHERE parent_id=$root"
    $parts = $rows -split '-'
    Assert-True ($parts[0] -eq $parts[1]) "存在重复实例：$rows"
}
Test-Case '单独完成某次实例不影响其他' {
    $root = (Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '逐次完成系列'; event_date = '2026-08-20'; start_time = '10:00:00'
        recurrence_type = 2; recurrence_rule = '{"weekdays":[1,3,5]}'
    }).Data.data.id
    $null = Invoke-Api GET '/todos/calendar?start_date=2026-08-20&end_date=2026-08-31' -Token $A.Token
    $cal24 = Invoke-Api GET '/todos/calendar?start_date=2026-08-24&end_date=2026-08-24' -Token $A.Token
    if ($cal24.Status -ne 200 -or $null -eq $cal24.Data.data -or $cal24.Data.data.Count -lt 1) {
        throw "8/24 日历查询异常：$($cal24.Status) $($cal24.Raw)"
    }
    $inst = $cal24.Data.data[0]
    $null = Invoke-Api PATCH "/todos/$($inst.id)/status" -Token $A.Token -Body @{ status = 1 }
    $again = (Invoke-Api GET '/todos/calendar?start_date=2026-08-24&end_date=2026-08-24' -Token $A.Token).Data.data[0]
    $again2 = (Invoke-Api GET '/todos/calendar?start_date=2026-08-26&end_date=2026-08-26' -Token $A.Token).Data.data[0]
    Assert-True ($again.status -eq 1) '实例完成状态未保留'
    Assert-True ($again2.status -eq 0) '其他实例受影响'
}

# ---------- 5. 记事 ----------
Write-Host ''
Write-Host '[记事]' -ForegroundColor Yellow
Test-Case '记事新建/列表/编辑/删除' {
    $r = Invoke-Api POST '/notes' -Token $A.Token -Body @{ title = '灵感笔记'; content = '记得买新耳机' }
    Assert-Status $r 200 '新建记事'
    $id = $r.Data.data.id
    $list = Invoke-Api GET '/notes' -Token $A.Token
    if ($list.Data.data.total -lt 1) { throw "记事列表为空，原始响应：$($list.Raw)" }
    $r2 = Invoke-Api PUT "/notes/$id" -Token $A.Token -Body @{ title = '灵感笔记2'; content = '已更新' }
    Assert-Status $r2 200 '编辑记事'
    $r3 = Invoke-Api DELETE "/notes/$id" -Token $A.Token
    Assert-Status $r3 200 '删除记事'
}
Test-Case '空标题记事被拒(400)' {
    $r = Invoke-Api POST '/notes' -Token $A.Token -Body @{ title = ' '; content = 'x' }
    Assert-Status $r 400 '空标题记事'
}

# ---------- 6. 习惯打卡 ----------
Write-Host ''
Write-Host '[习惯打卡]' -ForegroundColor Yellow
Test-Case '新建习惯/打卡/重复409/取消' {
    $h = (Invoke-Api POST '/habits' -Token $A.Token -Body @{ name = '喝水'; icon = '💧' }).Data.data.id
    $r1 = Invoke-Api POST "/habits/$h/checkin?date=2026-08-19" -Token $A.Token
    Assert-Status $r1 200 '打卡'
    $r2 = Invoke-Api POST "/habits/$h/checkin?date=2026-08-19" -Token $A.Token
    Assert-Status $r2 409 '重复打卡'
    $r3 = Invoke-Api DELETE "/habits/$h/checkin/2026-08-19" -Token $A.Token
    Assert-Status $r3 200 '取消打卡'
}
Test-Case '未来日期打卡被拒(400)' {
    $h = (Invoke-Api POST '/habits' -Token $A.Token -Body @{ name = '早睡' }).Data.data.id
    $r = Invoke-Api POST "/habits/$h/checkin?date=2026-08-25" -Token $A.Token
    Assert-Status $r 400 '未来打卡'
}
Test-Case '打卡早于习惯创建日期被拒(400)' {
    $h = (Invoke-Api POST '/habits' -Token $A.Token -Body @{ name = '新习惯' }).Data.data.id
    $r = Invoke-Api POST "/habits/$h/checkin?date=2020-01-01" -Token $A.Token
    Assert-Status $r 400 '早于创建日期打卡'
}
Test-Case '连续天数计算' {
    $h = (Invoke-Api POST '/habits' -Token $A.Token -Body @{ name = '连续习惯' }).Data.data.id
    $uid = (Invoke-Api GET '/auth/me' -Token $A.Token).Data.data.id
    $null = Sql-Query "INSERT INTO habit_records (habit_id,user_id,record_date,created_at) VALUES ($h,$uid,'2026-08-17',NOW()),($h,$uid,'2026-08-18',NOW())"
    $null = Invoke-Api POST "/habits/$h/checkin?date=2026-08-19" -Token $A.Token
    $r = Invoke-Api GET "/habits/$h/streak" -Token $A.Token
    Assert-True ($r.Data.data.streak -eq 3) "连续天数=$($r.Data.data.streak)，应为3"
}

# ---------- 7. 纪念日 ----------
Write-Host ''
Write-Host '[纪念日]' -ForegroundColor Yellow
Test-Case '新建纪念日倒计时正确' {
    $r = Invoke-Api POST '/anniversaries' -Token $A.Token -Body @{
        name = '结婚纪念日'; event_date = '2026-08-20'; repeat_yearly = $true
    }
    Assert-Status $r 200 '新建纪念日'
    if ($r.Data.data.days_left -ne 1) { throw "倒计时错误=$($r.Data.data.days_left)，原始响应：$($r.Raw)" }
}
Test-Case '已过纪念日按明年计算' {
    $r = Invoke-Api POST '/anniversaries' -Token $A.Token -Body @{
        name = '去年今天'; event_date = '2026-08-18'; repeat_yearly = $true
    }
    Assert-True ($r.Data.data.days_left -ge 0) '每年重复的纪念日不应为负'
}
Test-Case '纪念日提醒联动与删除清理' {
    $r = Invoke-Api POST '/anniversaries' -Token $A.Token -Body @{
        name = '待提醒纪念日'; event_date = '2026-08-25'; repeat_yearly = $true
        remind_enabled = $true; remind_days_before = 1
    }
    $id = $r.Data.data.id
    $tasks = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=2 AND target_id=$id"
    Assert-True ($tasks -eq '1') '纪念日提醒任务未生成'
    $null = Invoke-Api DELETE "/anniversaries/$id" -Token $A.Token
    $tasks = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=2 AND target_id=$id"
    Assert-True ($tasks -eq '0') '删除后提醒任务未清理'
}
Test-Case '他人纪念日越权(404)' {
    $B = New-TestUser
    $mine = (Invoke-Api POST '/anniversaries' -Token $B.Token -Body @{ name = 'B纪念日'; event_date = '2026-08-20' }).Data.data.id
    if ($null -eq $mine) { throw '创建纪念日返回的 id 为空' }
    $r = Invoke-Api GET "/anniversaries/$mine" -Token $A.Token
    Assert-Status $r 404 '越权纪念日'
}

# ---------- 8. 提醒联动 ----------
Write-Host ''
Write-Host '[提醒联动]' -ForegroundColor Yellow
Test-Case '待办提醒生成/重算/关闭' {
    $t = (Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '提醒联动'; event_date = '2026-08-21'; start_time = '14:00:00'
        reminder_enabled = $true; remind_offset_minutes = 10
    }).Data.data.id
    $row = Sql-Query "SELECT DATE_FORMAT(remind_at,'%Y-%m-%d %H:%i:%s') FROM reminder_tasks WHERE target_type=1 AND target_id=$t"
    Assert-True ($row -match '2026-08-21 13:50') "提醒时间错误: $row"
    $null = Invoke-Api PUT "/todos/$t" -Token $A.Token -Body @{
        title = '提醒联动'; event_date = '2026-08-21'; start_time = '16:00:00'
        reminder_enabled = $true; remind_offset_minutes = 10
    }
    $row = Sql-Query "SELECT DATE_FORMAT(remind_at,'%Y-%m-%d %H:%i:%s') FROM reminder_tasks WHERE target_type=1 AND target_id=$t"
    Assert-True ($row -match '2026-08-21 15:50') "提醒时间未重算: $row"
    $null = Invoke-Api PUT "/todos/$t" -Token $A.Token -Body @{
        title = '提醒联动'; event_date = '2026-08-21'; start_time = '16:00:00'
        reminder_enabled = $false
    }
    $cnt = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=1 AND target_id=$t"
    Assert-True ($cnt -eq '0') '关闭提醒后任务未删除'
}
Test-Case '重复事项每实例生成提醒且不重复' {
    $root = (Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '提醒系列'; event_date = '2026-08-20'; start_time = '10:00:00'
        recurrence_type = 1; reminder_enabled = $true; remind_offset_minutes = 5
    }).Data.data.id
    $null = Invoke-Api GET '/todos/calendar?start_date=2026-08-20&end_date=2026-08-23' -Token $A.Token
    $rows = Sql-Query "SELECT COUNT(*) FROM reminder_tasks WHERE target_type=1 AND target_id IN (SELECT id FROM todos WHERE parent_id=$root)"
    Assert-True ($rows -eq '3') "实例提醒数量=$rows，应为3（仅统计实例，根待办提醒另计）"
    $dup = Sql-Query "SELECT COUNT(*)-COUNT(DISTINCT CONCAT(target_type,'-',target_id)) FROM reminder_tasks WHERE target_type=1 AND target_id IN (SELECT id FROM todos WHERE parent_id=$root)"
    Assert-True ($dup -eq '0') '实例提醒存在重复'
}

# ---------- 9. 提醒送达（调度器+消费者） ----------
Write-Host ''
Write-Host '[提醒送达]' -ForegroundColor Yellow
Test-Case '到期提醒送达提醒中心且不重复' {
    # 开始时间设为 2 分钟后（今天的事项开始时间不能早于当前时间）
    $start = (Get-Date).AddMinutes(2).ToString('HH:mm:ss')
    $today = Get-Date -Format 'yyyy-MM-dd'
    $t = (Invoke-Api POST '/todos' -Token $A.Token -Body @{
        title = '立即提醒测试'; event_date = $today; start_time = $start
        reminder_enabled = $true; remind_offset_minutes = 0
    }).Data.data.id
    $taskStatus = Sql-Query "SELECT status FROM reminder_tasks WHERE target_type=1 AND target_id=$t"
    Assert-True ($null -ne $taskStatus -and $taskStatus -ne '') "提醒任务未创建"
    $deadline = (Get-Date).AddSeconds(150)
    $found = $false
    $lastRaw = ''
    $lastHit = -1
    while ((Get-Date) -lt $deadline) {
        Start-Sleep -Seconds 5
        $r = Invoke-Api GET '/notifications?page=1&page_size=50' -Token $A.Token
        $lastRaw = $r.Raw
        $hit = @($r.Data.data.list | Where-Object { $_.target_id -eq $t -and $_.target_type -eq 1 })
        $lastHit = $hit.Count
        if ($hit.Count -ge 1) {
            $found = $true
            break
        }
    }
    if (-not $found) {
        $st = Sql-Query "SELECT status FROM reminder_tasks WHERE target_type=1 AND target_id=$t"
        throw "150秒内提醒未送达提醒中心。t=$t 任务状态=$st 最后命中数=$lastHit，通知列表原始响应：$lastRaw"
    }
    $status = Sql-Query "SELECT status FROM reminder_tasks WHERE target_type=1 AND target_id=$t"
    Assert-True ($status -eq '2') "任务状态=$status，应为2（已送达）"
    $notifCount = Sql-Query "SELECT COUNT(*) FROM notifications WHERE target_type=1 AND target_id=$t"
    Assert-True ($notifCount -eq '1') "提醒记录=$notifCount，应为1"
    $n = (Invoke-Api GET '/notifications?page=1&page_size=50' -Token $A.Token).Data.data.list | Where-Object { $_.target_id -eq $t } | Select-Object -First 1
    $null = Invoke-Api PATCH "/notifications/$($n.id)/read" -Token $A.Token
}

# ---------- 10. 搜索与安全 ----------
Write-Host ''
Write-Host '[搜索与安全]' -ForegroundColor Yellow
Test-Case '跨模块搜索' {
    $r = Invoke-Api GET '/search?q=%E5%BC%80%E4%BC%9A' -Token $A.Token
    Assert-Status $r 200 '搜索'
    Assert-True ($r.Data.data.Count -ge 1) '搜索无结果'
}
Test-Case 'SQL注入字符串按普通文本处理' {
    $null = Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = "' OR 1=1 --"; event_date = '2026-08-20' }
    $r = Invoke-Api GET '/todos?keyword=%27%20OR%201%3D1%20--' -Token $A.Token
    Assert-Status $r 200 'SQL搜索'
    Assert-True ($r.Data.data.total -le 50) '疑似SQL注入生效'
}
Test-Case 'XSS字符串安全存储' {
    $null = Invoke-Api POST '/todos' -Token $A.Token -Body @{ title = '<script>alert(1)</script>'; event_date = '2026-08-20' }
    $r = Invoke-Api GET '/todos?keyword=script' -Token $A.Token
    Assert-True ($r.Data.data.total -ge 1) 'XSS字符串未存储'
}

# ---------- 汇总 ----------
Write-Host ''
Write-Host '================ 测试汇总 ================' -ForegroundColor Cyan
$pass = ($Results | Where-Object { $_.Result -eq 'PASS' }).Count
$fail = $Results.Count - $pass
$summaryColor = 'Green'
if ($fail -gt 0) { $summaryColor = 'Red' }
Write-Host "通过: $pass / $($Results.Count)" -ForegroundColor $summaryColor
if ($fail -gt 0) {
    Write-Host '失败明细：' -ForegroundColor Red
    $Results | Where-Object { $_.Result -ne 'PASS' } | ForEach-Object { Write-Host "  - $($_.Name): $($_.Result)" -ForegroundColor Red }
    exit 1
}
exit 0
