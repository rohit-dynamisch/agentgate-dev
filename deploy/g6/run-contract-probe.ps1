param(
    [switch]$ValidateEvidence = $false,
    [switch]$SkipDown = $false
)

$ErrorActionPreference = "Continue"
$PSScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$ComposeFile = Join-Path $PSScriptRoot "docker-compose.yml"

Write-Host "=== G6 Gateway Authorization Contract Probe ===" -ForegroundColor Cyan

# Step 1: Verify pinned image
Write-Host "Verifying pinned agentgateway image..."
& (Join-Path $PSScriptRoot "verify-image.ps1")

# Step 2: Bring up compose stack
Write-Host "Starting G6 Docker Compose topology..."
docker compose -f $ComposeFile down -v 2>$null
docker compose -f $ComposeFile up -d --build

try {
    # Wait for services to become healthy
    Write-Host "Waiting for services to become healthy..."
    $timeoutSeconds = 30
    $startTime = Get-Date

    while ((Get-Date) -lt $startTime.AddSeconds($timeoutSeconds)) {
        $authzHealth = (Invoke-RestMethod -Uri "http://localhost:9002/healthz" -ErrorAction SilentlyContinue)
        $mcpHealth = (Invoke-RestMethod -Uri "http://localhost:9101/healthz" -ErrorAction SilentlyContinue)
        if ($authzHealth -eq "OK" -and $mcpHealth -eq "OK") {
            break
        }
        Start-Sleep -Milliseconds 500
    }

    # Reset records & counters
    Invoke-RestMethod -Method Post -Uri "http://localhost:9002/reset" -ErrorAction SilentlyContinue | Out-Null
    Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" -ErrorAction SilentlyContinue | Out-Null

    # Step 3: Run probe-client tool call through gateway
    Write-Host "Sending real MCP initialize & tool call to gateway on :3000..."
    $env:GOTMPDIR = "d:\PROJECTS\AgentGate_Hackathon\agentgate-repo\.tmp"
    $clientDir = Join-Path $PSScriptRoot "probe-client"
    
    # Run probe-client directly
    $clientOut = & go run -C $clientDir . -url "http://localhost:3000" -tool "read_status" -skip-init 2>&1
    Write-Host "Probe client output: $clientOut"

    # Step 4: Gather evidence
    $recordsJson = Invoke-RestMethod -Uri "http://localhost:9002/records"
    $countJson = Invoke-RestMethod -Uri "http://localhost:9101/_g6/count"

    $probeCount = 0
    if ($recordsJson) {
        $probeCount = @($recordsJson).Count
    }
    $backendCount = 0
    if ($countJson -and $countJson.count -ne $null) {
        $backendCount = $countJson.count
    }

    Write-Host "Observed: ext_authz requests = $probeCount, backend invocations = $backendCount" -ForegroundColor Yellow

    # Step 4.1 assertion:
    if ($probeCount -ne 1 -or $backendCount -ne 1) {
        throw "G6 contract probe did not traverse exactly one authorization and backend call (probe: $probeCount, backend: $backendCount)"
    }

    Write-Host "SUCCESS: Real MCP call through agentgateway successfully authorized and executed on backend exactly once." -ForegroundColor Green

    # Save observed evidence artifact
    $evidenceDir = "$PSScriptRoot/../../docs/PHASES/G6_WORKSTREAMS"
    $evidenceFile = Join-Path $evidenceDir "G6_GATEWAY_CONTRACT_OBSERVED.json"
    $evidenceObj = @{
        timestamp = (Get-Date).ToUniversalTime().ToString("o")
        probe_records = $recordsJson
        backend_evidence = $countJson
        client_output = "$clientOut"
    }
    $evidenceObj | ConvertTo-Json -Depth 10 | Set-Content -Path $evidenceFile -Encoding UTF8
    Write-Host "Observed contract evidence saved to: $evidenceFile"

    if ($ValidateEvidence) {
        Write-Host "Validating complete contract evidence and fail-closed properties..."
        $firstRec = @($recordsJson)[0]
        $hasBody = $firstRec.body_present -eq $true -and $firstRec.raw_body -match "read_status"
        $notTruncated = $firstRec.body_length -gt 0 -and $firstRec.raw_body.Length -ge $firstRec.body_length
        $backendOk = $countJson.count -eq 1 -and @($countJson.invocations)[0].tool_name -eq "read_status"
        $hasMethodPath = $firstRec.method -eq "POST" -and $firstRec.path -ne $null

        if (-not $hasBody) { throw "unsafe missing contract fact: body_complete" }
        if (-not $notTruncated) { throw "unsafe missing contract fact: not_truncated" }
        if (-not $backendOk) { throw "unsafe missing contract fact: route_provenance" }
        if (-not $hasMethodPath) { throw "unsafe missing contract fact: identity_provenance" }

        # --- Negative Scenario 1: Mode = DENY ---
        Write-Host "Testing fail-closed invariant: DENY mode..."
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/mode?mode=deny" | Out-Null
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/reset" | Out-Null
        Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" | Out-Null
        $denyOut = & go run -C $clientDir . -url "http://localhost:3000" -tool "read_status" -skip-init 2>&1
        $denyBackendCount = (Invoke-RestMethod -Uri "http://localhost:9101/_g6/count").count
        if ($denyBackendCount -ne 0) {
            throw "SECURITY DEFECT: DENY mode reached backend (backendCount: $denyBackendCount)"
        }
        Write-Host "PASS: DENY mode resulted in 0 backend invocations." -ForegroundColor Green

        # --- Negative Scenario 2: Mode = MALFORMED ---
        Write-Host "Testing fail-closed invariant: MALFORMED mode..."
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/mode?mode=malformed" | Out-Null
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/reset" | Out-Null
        Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" | Out-Null
        $malformedOut = & go run -C $clientDir . -url "http://localhost:3000" -tool "read_status" -skip-init 2>&1
        $malformedBackendCount = (Invoke-RestMethod -Uri "http://localhost:9101/_g6/count").count
        if ($malformedBackendCount -ne 0) {
            throw "SECURITY DEFECT: MALFORMED mode reached backend (backendCount: $malformedBackendCount)"
        }
        Write-Host "PASS: MALFORMED mode resulted in 0 backend invocations." -ForegroundColor Green

        # --- Negative Scenario 3: Mode = UNAVAILABLE ---
        Write-Host "Testing fail-closed invariant: UNAVAILABLE mode..."
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/mode?mode=unavailable" | Out-Null
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/reset" | Out-Null
        Invoke-RestMethod -Method Post -Uri "http://localhost:9101/_g6/reset" | Out-Null
        $unavailOut = & go run -C $clientDir . -url "http://localhost:3000" -tool "read_status" -skip-init 2>&1
        $unavailBackendCount = (Invoke-RestMethod -Uri "http://localhost:9101/_g6/count").count
        if ($unavailBackendCount -ne 0) {
            throw "SECURITY DEFECT: UNAVAILABLE mode reached backend (backendCount: $unavailBackendCount)"
        }
        Write-Host "PASS: UNAVAILABLE mode resulted in 0 backend invocations." -ForegroundColor Green

        # Reset probe to allow mode
        Invoke-RestMethod -Method Post -Uri "http://localhost:9002/mode?mode=allow" | Out-Null

        Write-Host "ALL EVIDENCE AND FAIL-CLOSED PROPERTIES VERIFIED SUCCESSFULLY." -ForegroundColor Green
    }
}
finally {
    if (-not $SkipDown) {
        Write-Host "Cleaning up G6 topology..."
        cmd /c "docker compose -f $ComposeFile down -v 2>nul"
    }
}
