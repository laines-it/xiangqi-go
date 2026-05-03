$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$webDir = Join-Path $repoRoot "web"

if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache-codex"
}

$env:GOOS = "js"
$env:GOARCH = "wasm"

go build -buildvcs=false -o (Join-Path $webDir "godogpaw.wasm") ./cmd/wasm
Copy-Item -Force -LiteralPath (Join-Path (go env GOROOT) "lib\wasm\wasm_exec.js") -Destination (Join-Path $webDir "wasm_exec.js")

Write-Host "Built web/godogpaw.wasm and web/wasm_exec.js"
