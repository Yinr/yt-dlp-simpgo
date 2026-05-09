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

# 获取版本号
$Version = git describe --tags --always --dirty 2>$null
if (-not $Version) { $Version = "dev" }

Write-Host "Installing modules..."
go mod tidy

$Ldflags = "-X 'main.Version=$Version' -s -w"

if ($Gui) {
    Write-Host "Generating Windows icon resource..."
    Remove-Item -LiteralPath "rsrc.syso" -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "rsrc_windows_amd64.syso" -Force -ErrorAction SilentlyContinue
    go run github.com/akavel/rsrc@v0.10.2 -ico "res/icon.ico" -arch amd64 -o "rsrc_windows_amd64.syso"

    Write-Host "Building GUI exe -> $OutPath (windowsgui) version=$Version"
    $cmd = "go build -v -ldflags `"$Ldflags -H=windowsgui`" -o `"$OutPath`" $Pkg"
} else {
    Write-Host "Building console exe -> $OutPath version=$Version"
    $cmd = "go build -v -ldflags `"$Ldflags`" -o `"$OutPath`" $Pkg"
}

Write-Host $cmd
Invoke-Expression $cmd

Pop-Location
