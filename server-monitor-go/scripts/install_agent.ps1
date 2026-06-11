<#
.SYNOPSIS
  Install the Go monitoring agent as a Windows service.

.DESCRIPTION
  Downloads the static agent binary from the monitoring hub and registers it
  as an auto-starting Windows service. Run in an elevated (Administrator)
  PowerShell.

.PARAMETER NodeName
  Must match a client registered in the dashboard (Admin -> Add Client).

.PARAMETER ServerUrl
  Base URL of the hub, e.g. http://100.98.88.100:5000

.EXAMPLE
  .\install_agent.ps1 -NodeName win-1 -ServerUrl http://100.98.88.100:5000
#>
param(
  [Parameter(Mandatory = $true)][string]$NodeName,
  [Parameter(Mandatory = $true)][string]$ServerUrl
)

$ErrorActionPreference = 'Stop'
$ServerUrl = $ServerUrl.TrimEnd('/')

$installDir = 'C:\server-monitor-agent'
$binPath    = Join-Path $installDir 'monitor-agent.exe'
$binUrl     = "$ServerUrl/download/monitor-agent-windows-amd64.exe"
$serviceName = 'ServerMonitorAgent'

Write-Host "Installing monitoring agent" -ForegroundColor Green
Write-Host "  node name : $NodeName"
Write-Host "  hub       : $ServerUrl"

New-Item -ItemType Directory -Force -Path $installDir | Out-Null

Write-Host "Downloading agent from $binUrl ..."
Invoke-WebRequest -Uri $binUrl -OutFile $binPath

# Recreate the service if it already exists.
if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
  Stop-Service  $serviceName -ErrorAction SilentlyContinue
  sc.exe delete $serviceName | Out-Null
  Start-Sleep -Seconds 1
}

$bin = "`"$binPath`" --name `"$NodeName`" --server `"$ServerUrl`" --interval 2s"
New-Service -Name $serviceName -BinaryPathName $bin `
  -DisplayName 'Distributed Server Monitor Agent' -StartupType Automatic | Out-Null
Start-Service $serviceName

Write-Host "Done. Service '$serviceName' is running and reporting '$NodeName'." -ForegroundColor Green
Write-Host "Manage: Get-Service $serviceName | Restart-Service $serviceName | Stop-Service $serviceName" -ForegroundColor Yellow
