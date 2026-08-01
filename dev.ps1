# 开发模式一键启动脚本
# 启动 Go 后端 + Vite 前端开发服务器
# 用法：在 PowerShell 中直接运行 .\dev.ps1
#       Ctrl+C 停止所有服务

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path

# ---- 1. 检查前端依赖 ----
$nodeModules = [System.IO.Path]::Combine($ProjectRoot, "web", "node_modules")
if (-not (Test-Path $nodeModules)) {
    Write-Host "⏳ 安装前端依赖..." -ForegroundColor Cyan
    Push-Location (Join-Path $ProjectRoot "web")
    try {
        npm install
        if ($LASTEXITCODE -ne 0) { throw "npm install failed" }
    } finally {
        Pop-Location
    }
    Write-Host "✓ 前端依赖安装完成" -ForegroundColor Green
}

# ---- 2. 启动后端（隐藏窗口，日志写入文件） ----
# 注意：Start-Process 不允许 stdout/stderr 重定向到同一文件，必须分开。
$dataDir = Join-Path $ProjectRoot ".harness-data"
$null = New-Item -ItemType Directory -Force -Path $dataDir
$backendOutLog = Join-Path $dataDir "backend.out.log"
$backendErrLog = Join-Path $dataDir "backend.err.log"

Write-Host "⏳ 启动 Go 后端 (127.0.0.1:8000)..." -ForegroundColor Cyan

$backendProc = Start-Process -WindowStyle Hidden -PassThru `
    -FilePath "go" -ArgumentList "run", "./cmd/server/" `
    -WorkingDirectory $ProjectRoot `
    -RedirectStandardOutput $backendOutLog -RedirectStandardError $backendErrLog

try {
    # 等待后端就绪（最多 15 秒）
    $ready = $false
    for ($i = 0; $i -lt 30; $i++) {
        Start-Sleep -Milliseconds 500
        try {
            $response = Invoke-WebRequest -Uri "http://127.0.0.1:8000/health" -UseBasicParsing -TimeoutSec 2
            if ($response.StatusCode -eq 200) { $ready = $true; break }
        } catch {
            # 服务尚未就绪，继续等待（PowerShell 没有 C 风格注释）
        }
    }

    if (-not $ready) {
        Write-Host "✗ 后端启动超时，请检查日志:" -ForegroundColor Red
        foreach ($log in @($backendErrLog, $backendOutLog)) {
            if (Test-Path $log) { Get-Content $log -Tail 10 }
        }
        # exit 会触发外层 finally，统一走整树清理
        exit 1
    }
    Write-Host "✓ 后端已就绪 (http://127.0.0.1:8000)" -ForegroundColor Green

    # ---- 3. 启动前端开发服务器（前台，Ctrl+C 可停止） ----
    # 注意：Ctrl+C 后 finally 清理依赖 PowerShell 7+ 行为；Windows PowerShell 5.1 不保证执行。
    Write-Host @"

╔══════════════════════════════════════════════════════════════════╗
║  浏览器打开: http://127.0.0.1:48623                              ║
║  后端日志:    $backendOutLog  ║
║  错误日志:    $backendErrLog  ║
║  按 Ctrl+C 停止所有服务（推荐 PowerShell 7+）                    ║
╚══════════════════════════════════════════════════════════════════╝

"@ -ForegroundColor Cyan

    Push-Location (Join-Path $ProjectRoot "web")
    try {
        # 直接运行 npx，Ctrl+C 会正确终止并执行 finally 块（PS 7+）
        npx.cmd vite --host 127.0.0.1 --port 48623
    } finally {
        Pop-Location
    }
} finally {
    # 停止后端（/T 杀整棵进程树，避免 go run 派生的 server 子进程残留）
    if ($backendProc -and !$backendProc.HasExited) {
        Write-Host "⏳ 停止后端服务..." -ForegroundColor Cyan
        & taskkill /PID $backendProc.Id /T /F 2>$null | Out-Null
        $backendProc.WaitForExit(5000)
    }
    Write-Host "✓ 已停止" -ForegroundColor Green
}
