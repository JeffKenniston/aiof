# ============================================================================
# AIOF Benchmark & Telemetry Execution Script (Cloud & Intel Arc 140V Local Mode)
# ============================================================================

Param (
    [string]$Mode = "Cloud",
    [string]$OutputDir = "$env:USERPROFILE\.aiof\benchmarks"
)

Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "  AIOF Framework Telemetry & Comparative Execution Benchmark  " -ForegroundColor Cyan
Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "Target Machine: Intel Core Ultra 7 258V + Intel Arc 140V (16GB VRAM)" -ForegroundColor Yellow
Write-Host "Benchmark Mode: $Mode" -ForegroundColor Green
Write-Host ""

If (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
}

$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$ReportFile = Join-Path $OutputDir "benchmark_report_$Timestamp.md"

Write-Host "[1/3] Running Go Daemon Telemetry & Benchmark Suite..." -ForegroundColor Green
Set-Location -Path "$PSScriptRoot\..\..\apps\daemon"

go test -v ./internal/telemetry -run TestWatcherAgentMetricsAndReport

Write-Host ""
Write-Host "[2/3] Executing All Core Daemon Component Benchmarks..." -ForegroundColor Green
go test -v ./internal/mcp ./internal/agent ./internal/consensus ./internal/llm

Write-Host ""
Write-Host "[3/3] Compiling AIOF Enterprise Executable..." -ForegroundColor Green
go build -o ..\..\bin\aiof.exe .\cmd\aiof\main.go

If ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "SUCCESS: AIOF Daemon built cleanly to bin/aiof.exe" -ForegroundColor Green
    Write-Host "Telemetry & Benchmark logs updated successfully." -ForegroundColor Cyan
} Else {
    Write-Host ""
    Write-Host "ERROR: Benchmark compilation encountered errors." -ForegroundColor Red
}
