@echo off
setlocal enabledelayedexpansion

REM Configuration
set "REPO=DazFather/Wednesday"
set "API=https://api.github.com/repos/%REPO%/releases/latest"

REM Detect architecture
set "ARCH=amd64"
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

REM Check for PowerShell (needed only to get latest version and unzip)
where powershell >nul 2>&1 || (
    echo Error: PowerShell is required to run this installer.
    exit /b 1
)

REM Fetch latest version tag from GitHub
for /f "delims=" %%i in ('powershell -NoProfile -Command "(Invoke-RestMethod '%API%').tag_name"') do (
    set "VERSION=%%i"
)

if not defined VERSION (
    echo Failed to fetch latest version tag!
    exit /b 1
)

echo Latest version: %VERSION%

REM Compose download URL and ZIP filename
set "FILENAME=wed-windows-%ARCH%.zip"
set "URL=https://github.com/%REPO%/releases/download/%VERSION%/%FILENAME%"

REM Create a temp dir
set "TMP_DIR=%TEMP%\wed_installer_%RANDOM%"
mkdir "%TMP_DIR%" || exit /b 1

REM Download the archive
echo Downloading %FILENAME%...
powershell -NoProfile -Command "Invoke-WebRequest -Uri '%URL%' -OutFile '%TMP_DIR%\%FILENAME%'" || (
    echo Failed to download the archive!
    exit /b 1
)

REM Extract the ZIP
echo Extracting...
powershell -NoProfile -Command "Expand-Archive -Path '%TMP_DIR%\%FILENAME%' -DestinationPath '%TMP_DIR%' -Force" || (
    echo Failed to extract archive!
    exit /b 1
)

REM Run install.bat
if exist "%TMP_DIR%\install.bat" (
    echo Running installer...
    call "%TMP_DIR%\install.bat"
) else (
    echo install.bat not found in archive!
    exit /b 1
)

REM Cleanup
rmdir /s /q "%TMP_DIR%" >nul 2>&1

echo Installed successfully!
exit /b 0

