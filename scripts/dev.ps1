param([ValidateSet("init","up","down","logs","check","smoke","migrate")][string]$Task = "up")
$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)
function Run { param([string]$Program,[string[]]$Arguments) & $Program @Arguments; if ($LASTEXITCODE -ne 0) { throw "$Program failed ($LASTEXITCODE)" } }
switch ($Task) {
 "init" {
  if (Test-Path .env) { Write-Output ".env already exists; preserved."; break }
  $content = Get-Content .env.example -Raw
  foreach ($key in @("POSTGRES_PASSWORD","REDIS_PASSWORD","TURN_SECRET")) {
   $bytes = New-Object byte[] 32
   $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
   try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
   $secret = -join ($bytes | ForEach-Object { $_.ToString("x2") })
   $content = [regex]::Replace($content, "(?m)^" + $key + "=.*$", $key + "=" + $secret)
  }
  [IO.File]::WriteAllText((Join-Path (Get-Location) ".env"),$content,[Text.UTF8Encoding]::new($false))
  Write-Output "Created .env with unique local secrets."
 }
 "up" { Run docker @("compose","up","--build","--detach","--wait","--wait-timeout","180") }
 "down" { Run docker @("compose","down") }
 "logs" { Run docker @("compose","logs","--tail","100","--follow") }
 "migrate" { Run docker @("compose","run","--rm","migrate") }
 "check" {
  Run go @("test","./...")
  Run go @("vet","./...")
  Push-Location apps/web
  try { Run npm.cmd @("run","typecheck"); Run npm.cmd @("run","lint"); Run npm.cmd @("run","build") } finally { Pop-Location }
 }
 "smoke" { Run node @("scripts/smoke.mjs") }
}
