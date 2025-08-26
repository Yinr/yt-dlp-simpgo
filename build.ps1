<#
Simple build helper for this project.

Usage:
  .\build.ps1                # build normal executable
  .\build.ps1 -Gui          # build windows GUI (adds -ldflags "-H=windowsgui")
  .\build.ps1 -Out out.exe  # specify output name
#>

param(
    [switch]$Gui,
    [string]$Out = "yt-dlp-simpgo.exe",
    [string]$Pkg = "."
)

Push-Location $PSScriptRoot

Write-Host "Installing modules..."
go mod tidy

$argsList = @()
if ($Gui) {
    Write-Host "Building GUI exe (windowsgui)..."
    $cmd = "go build -v -ldflags `"-H=windowsgui`" -o `"$Out`" $Pkg"
} else {
    Write-Host "Building console exe..."
    $cmd = "go build -v -o `"$Out`" $Pkg"
}

Write-Host $cmd
Invoke-Expression $cmd

Pop-Location
