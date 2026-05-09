<#
构建脚本，输出到 dist/ 目录。

用法:
  .\build.ps1                # 构建控制台版本
  .\build.ps1 -Gui           # 构建 Windows GUI 版本（隐藏控制台）
  .\build.ps1 -Out myapp.exe # 指定输出文件名
  .\build.ps1 -Clean         # 清理构建产物
  .\build.ps1 -CleanRuntime  # 清理运行时文件
  .\build.ps1 -CleanAll      # 清理构建产物和运行时文件
#>

param(
    [switch]$Gui,
    [switch]$Clean,
    [switch]$CleanRuntime,
    [switch]$CleanAll,
    [string]$Out = "yt-dlp-simpgo.exe",
    [string]$Pkg = "."
)

Push-Location $PSScriptRoot

function Remove-BuildArtifacts {
    Remove-Item -LiteralPath "dist" -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "rsrc.syso" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "rsrc_windows_*.syso" -Force -ErrorAction SilentlyContinue
}

function Remove-RuntimeFiles {
    Remove-Item -LiteralPath "yt-dlp-simpgo.ini" -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "yt-dlp.conf" -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "yt-dlp.exe" -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "yt-dlp" -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "下载" -Recurse -Force -ErrorAction SilentlyContinue
}

if ($CleanAll) {
    Remove-BuildArtifacts
    Remove-RuntimeFiles
    Pop-Location
    return
}

if ($Clean) {
    Remove-BuildArtifacts
    Pop-Location
    return
}

if ($CleanRuntime) {
    Remove-RuntimeFiles
    Pop-Location
    return
}

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
