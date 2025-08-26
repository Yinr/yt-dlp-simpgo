param(
    [string]$Output = "yt-dlp-simpgo.exe",
    [string]$Pkg = "."
)

# change to repo root (scripts sits in scripts/)
Push-Location (Split-Path -Parent $PSScriptROOT)

Write-Host "Running: go mod tidy"
go mod tidy

Write-Host "Building Windows GUI exe -> $Output (hides console window)"
& go build -v -ldflags "-H=windowsgui" -o $Output $Pkg

Pop-Location
