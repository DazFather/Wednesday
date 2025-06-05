$ErrorActionPreference = "Stop"

function Write-Info($msg) { Write-Host $msg -ForegroundColor Cyan }
function Write-Success($msg) { Write-Host $msg -ForegroundColor Green }
function Write-ErrorMsg($msg) { Write-Host $msg -ForegroundColor Red }

$repo = "DazFather/Wednesday"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

try {
    $version = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
} catch {
    Write-ErrorMsg "Failed to retrieve latest version from GitHub: $_"
    exit 1
}
Write-Info "Latest version: $version"

$filename = "wed-windows-$arch.zip"
$downloadUrl = "https://github.com/$repo/releases/download/$version/$filename"

$tmpDir = Join-Path $env:TEMP "wed_installer_$([System.Guid]::NewGuid().ToString())"
New-Item -ItemType Directory -Path $tmpDir | Out-Null
Write-Info "Temp directory: $tmpDir"

$zipPath = Join-Path $tmpDir $filename
Write-Info "Downloading $filename..."
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath
} catch {
    Write-ErrorMsg "Download failed: $_"
    exit 1
}
Write-Success "Download completed: $zipPath"

Write-Info "Extracting archive..."
Expand-Archive -LiteralPath $zipPath -DestinationPath $tmpDir -Force
Write-Success "Extraction completed."

Write-Info "Looking for install.bat..."
$installer = Get-ChildItem -Path $tmpDir -Filter "install.bat" -Recurse | Select-Object -First 1

if (-not $installer) {
    Write-ErrorMsg "install.bat not found."
    exit 1
}
Write-Success "Found: $($installer.FullName)"

Write-Info "Starting installation..."
$prevDir = Get-Location
Set-Location $tmpDir

$output = & cmd.exe /c "`"$($installer.FullName)`""
Write-Output $output
if ($LASTEXITCODE -ne 0) {
    Write-ErrorMsg "Installer exited with code $LASTEXITCODE"
    Set-Location $prevDir
    exit $LASTEXITCODE
} else {
    Write-Success "Installation completed."
    Set-Location $prevDir
    Write-Info "Cleaning up temporary files..."
    Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

