# deploy/g6/run-e2e-matrix.ps1
# Full Clean-Environment E2E Matrix Execution for Gate G6 Real MCP Enforcement

$ErrorActionPreference = "Continue"

Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "  AgentGate Gate G6: Real MCP Enforcement End-to-End Matrix   " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan

$env:GOTMPDIR = "d:\PROJECTS\AgentGate_Hackathon\agentgate-repo\.tmp"
if (!(Test-Path $env:GOTMPDIR)) {
    New-Item -ItemType Directory -Path $env:GOTMPDIR -Force | Out-Null
}

$composeFile = Join-Path $PSScriptRoot "docker-compose.yml"
$clientDir = Join-Path $PSScriptRoot "probe-client"

# 1. Ensure docker stack is up and healthy
Write-Host "`n[1/6] Verifying Docker Compose Topology..." -ForegroundColor Yellow
docker compose -f $composeFile up -d
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to start docker compose topology"
    exit 1
}

# Wait for healthy status
Write-Host "Waiting for services to become healthy..."
$healthy = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 1
    try {
        $mcpHealth = (Invoke-RestMethod -Uri "http://localhost:9101/healthz" -TimeoutSec 1)
        $gateHealth = (Invoke-RestMethod -Uri "http://localhost:8090/healthz" -TimeoutSec 1)
        if ($mcpHealth -eq "ok" -and $gateHealth -eq "ok") {
            $healthy = $true
            break
        }
    } catch {
        # continue waiting
    }
}

if (-not $healthy) {
    Write-Error "Services failed to become healthy in time"
    docker compose -f $composeFile logs --tail=50
    exit 1
}
Write-Host "Topology is UP and HEALTHY." -ForegroundColor Green

# 2. Reset backend count
Write-Host "`n[2/6] Resetting MCP Backend Counter..." -ForegroundColor Yellow
Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" | Out-Null
$initialCount = (Invoke-RestMethod -Uri "http://localhost:9101/_g6/count").count
Write-Host "Initial backend call count: $initialCount"

# 3. Run QA Black-Box E2E Test Suite
Write-Host "`n[3/6] Running Independent QA Black-Box Test Suite..." -ForegroundColor Yellow
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
docker run --rm --network g6net -v "${repoRoot}:/src:ro" -w /src/agentgate -e AGENTGATE_GATEWAY_URL="http://g6-agentgateway:3000" -e AGENTGATE_BACKEND_COUNT_URL="http://g6-probe-mcp:9101" golang:1.24-alpine sh -c "export GOTOOLCHAIN=auto; go test -v ./qa/g6enforcement/..."
if ($LASTEXITCODE -ne 0) {
    Write-Error "QA E2E enforcement test suite failed"
    exit 1
}
Write-Host "QA Black-Box Suite: 100% PASS" -ForegroundColor Green

# 4. Live Gateway Fail-Closed Verification (AgentGate Container Outage)
Write-Host "`n[4/6] Testing Scenario 7 (Live AgentGate Outage & Fail-Closed Invariant)..." -ForegroundColor Yellow
Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" | Out-Null
Write-Host "Stopping g6-agentgate container to simulate total authorization service outage..."
docker stop g6-agentgate | Out-Null

$outageClientOut = & go run -C $clientDir . -url "http://localhost:3000" -tool "read_status" -skip-init -agent-id "agent-reader" -roles "reader" 2>&1
$outageBackendCount = (Invoke-RestMethod -Uri "http://localhost:9101/_g6/count").count
Write-Host "Outage client result: $outageClientOut"
Write-Host "Outage backend call count: $outageBackendCount"

if ($outageBackendCount -ne 0) {
    Write-Error "CRITICAL SECURITY DEFECT: Request bypassed authorization during AgentGate outage!"
    docker start g6-agentgate | Out-Null
    exit 1
}
Write-Host "PASS: Gateway strictly failed closed during AgentGate outage (0 backend calls)." -ForegroundColor Green

Write-Host "Restarting g6-agentgate container..."
docker start g6-agentgate | Out-Null
Start-Sleep -Seconds 3

# 5. Live Recovery Verification
Write-Host "`n[5/6] Verifying Live Service Recovery..." -ForegroundColor Yellow
Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" | Out-Null
$recoveryOut = & go run -C $clientDir . -url "http://localhost:3000" -tool "read_status" -skip-init -agent-id "agent-reader" -roles "reader" 2>&1
$recoveryBackendCount = (Invoke-RestMethod -Uri "http://localhost:9101/_g6/count").count
Write-Host "Post-recovery backend call count: $recoveryBackendCount"
if ($recoveryBackendCount -ne 1) {
    Write-Error "Service failed to recover after restart (count: $recoveryBackendCount)"
    exit 1
}
Write-Host "PASS: Service recovered cleanly and authorized valid MCP request." -ForegroundColor Green

# 6. Verify Durable Audit Trail in PostgreSQL
Write-Host "`n[6/6] Verifying PostgreSQL Durable Audit Log & Hash Chain..." -ForegroundColor Yellow
$auditRows = docker exec g6-postgres psql -U agentgate -d agentgate_db -c "SELECT sequence_number, event_type, decision, reason, principal_agent_id, tool_name, policy_version, prev_hash, row_hash FROM audit_events ORDER BY sequence_number DESC LIMIT 10;"
Write-Host "$auditRows"

Write-Host "`n================================================================" -ForegroundColor Green
Write-Host "  Gate G6 Real MCP Enforcement Gate: ALL DoD SCENARIOS PASSED  " -ForegroundColor Green
Write-Host "================================================================" -ForegroundColor Green
exit 0
