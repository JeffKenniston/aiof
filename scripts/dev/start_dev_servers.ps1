# ============================================================================
# AIOF Dev Server Launcher with Watcher Telemetry Agent Attached
# ============================================================================

Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "  Launching AIOF Dev Servers & Attaching Watcher Telemetry Agent  " -ForegroundColor Cyan
Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host ""

# 1. Kill any existing dev server processes
Write-Host "[1/3] Stopping any running AIOF / Vite processes..." -ForegroundColor Yellow
Stop-Process -Name "aiof", "node", "vite" -ErrorAction SilentlyContinue

# 2. Launch Vite Frontend Server in apps/web
Write-Host "[2/3] Starting Vite Frontend Server (@aiof/web)..." -ForegroundColor Green
Start-Process -FilePath "pnpm" -ArgumentList "--filter", "@aiof/web", "dev" -WorkingDirectory "$PSScriptRoot\..\.." -NoNewWindow

# 3. Launch Go Daemon Server with Watcher Telemetry Agent
Write-Host "[3/3] Starting AIOF Go Daemon Server (apps/daemon) with Watcher Agent..." -ForegroundColor Green
Set-Location -Path "$PSScriptRoot\..\..\apps\daemon"
$env:AIOF_WATCHER_TELEMETRY="true"
$env:GEMINI_API_KEY="$env:GEMINI_API_KEY"

go run ./cmd/aiof/main.go start
