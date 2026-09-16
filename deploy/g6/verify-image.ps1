param(
    [string]$ManifestPath = "$PSScriptRoot/IMAGE_DIGESTS.md",
    [string]$ImageRef = "cr.agentgateway.dev/agentgateway:v1.4.0"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $ManifestPath)) {
    throw "Manifest file not found: $ManifestPath"
}

$content = Get-Content -Raw $ManifestPath
$match = [regex]::Match($content, 'sha256:[a-f0-9]{64}')
if (-not $match.Success) {
    throw "No sha256 digest found in manifest: $ManifestPath"
}
$expectedDigest = $match.Value

$actualRepoDigests = docker image inspect $ImageRef --format '{{json .RepoDigests}}' 2>$null
if ($LASTEXITCODE -ne 0 -or -not $actualRepoDigests) {
    throw "Failed to inspect image $ImageRef. Is Docker running and image pulled?"
}

if ($actualRepoDigests -notmatch [regex]::Escape($expectedDigest)) {
    throw "agentgateway digest mismatch: expected $expectedDigest in $actualRepoDigests"
}

Write-Output "agentgateway pinned digest verified: $expectedDigest"
