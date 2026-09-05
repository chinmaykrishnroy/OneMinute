$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)
$settings = @{}
Get-Content .env | ForEach-Object {
 if ($_ -match '^([A-Z_]+)=(.*)$') { $settings[$Matches[1]] = $Matches[2].Trim() }
}
$env:TEST_DATABASE_URL = "postgres://$($settings.POSTGRES_USER):$($settings.POSTGRES_PASSWORD)@localhost:$($settings.POSTGRES_PORT)/$($settings.POSTGRES_DB)?sslmode=disable"
$env:TEST_REDIS_URL = "redis://:$($settings.REDIS_PASSWORD)@localhost:$($settings.REDIS_PORT)/0"
$env:TEST_TURN_ADDR = "127.0.0.1:$($settings.TURN_PORT)"
foreach ($key in @("TURN_SECRET","TURN_REALM","TURN_RELAY_MIN","TURN_RELAY_MAX")) { [Environment]::SetEnvironmentVariable($key,$settings[$key],"Process") }
go test -tags integration -count=1 -timeout 90s -v ./apps/server/internal/migrations ./apps/server/internal/signaling ./services/turn
if ($LASTEXITCODE -ne 0) { throw "Integration tests failed" }
