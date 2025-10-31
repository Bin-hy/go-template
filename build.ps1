# =============================
# 配置
# =============================
$projectName = "file_system"         # 项目名
$entryFile = "main.go"         # 入口文件

# =============================
# 创建输出目录 dist/YYYY-MM-DD/PROJECT_NAME
# =============================
$date = Get-Date -Format "yyyy-MM-dd"
$outputDir = Join-Path -Path "dist" -ChildPath "$date\$projectName"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

# =============================
# 禁用 CGO
# =============================
$env:CGO_ENABLED="0"

# =============================
# 目标平台列表
# =============================
$platforms = @(
    @{GOOS="windows"; GOARCH="amd64"; EXT=".exe"},
    @{GOOS="windows"; GOARCH="386"; EXT=".exe"},
    @{GOOS="linux";   GOARCH="amd64"; EXT=""},
    @{GOOS="linux";   GOARCH="386"; EXT=""},
    @{GOOS="darwin";  GOARCH="amd64"; EXT=""},
    @{GOOS="darwin";  GOARCH="arm64"; EXT=""}
)

# =============================
# 循环编译
# =============================
foreach ($p in $platforms) {
    $env:GOOS = $p.GOOS
    $env:GOARCH = $p.GOARCH

    $ext = $p.EXT
    $outputFile = Join-Path $outputDir ("$projectName" + "_" + "$($p.GOOS)" + "_" + "$($p.GOARCH)$ext")

    Write-Host "Compiling $($p.GOOS)_$($p.GOARCH) -> $outputFile"
    go build -o $outputFile $entryFile

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Failed to build $($p.GOOS)_$($p.GOARCH)" -ForegroundColor Red
    } else {
        Write-Host "✅ Success"
    }
}

Write-Host "=============================="
Write-Host "All builds completed! Output dir: $outputDir"
Write-Host "=============================="
