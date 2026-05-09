<#
构建脚本，输出到 dist/ 目录。

用法:
  .\build.ps1                # 构建控制台版本
  .\build.ps1 -Gui           # 构建 Windows GUI 版本（隐藏控制台）
  .\build.ps1 -Out myapp.exe # 指定输出文件名
#>

param(
    [switch]$Gui,
    [string]$Out = "yt-dlp-simpgo.exe",
    [string]$Pkg = "."
)

Push-Location $PSScriptRoot

$distDir = "dist"
if (-not (Test-Path $distDir)) {
    New-Item -ItemType Directory -Path $distDir | Out-Null
}

$OutPath = Join-Path $distDir $Out

Write-Host "Installing modules..."
go mod tidy

if ($Gui) {
    Write-Host "Building GUI exe -> $OutPath (windowsgui)"
    $cmd = "go build -v -ldflags `"-H=windowsgui`" -o `"$OutPath`" $Pkg"
} else {
    Write-Host "Building console exe -> $OutPath"
    $cmd = "go build -v -o `"$OutPath`" $Pkg"
}

Write-Host $cmd
Invoke-Expression $cmd

Pop-Location
