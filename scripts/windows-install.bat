@echo off
:: Simplified Windows installer
setlocal enabledelayedexpansion

:: -- Version detection --
git describe --always --tags 2>nul >nul
if %errorlevel% equ 0 (
    for /f "delims=" %%G in ('git describe --always --tags') do set "wed_version=%%G"
) else (
    for /f "delims=" %%G in ('git rev-parse --short HEAD') do set "wed_version=%%G"
)

:: -- Binary directory logic (simplified if/else) --
if not "%GOBIN%"=="" (
    set "install_dir=%GOBIN%"
) else if not "%GOPATH%"=="" (
    set "install_dir=%GOPATH%\bin"
) else if exist "%USERPROFILE%\go\bin\" (
    set "install_dir=%USERPROFILE%\go\bin"
) else (
    :: Default fallback: Create nested dirs in %APPDATA%
    set "install_dir=%APPDATA%\wed\bin"
    if not exist "%install_dir%" (
        mkdir "%APPDATA%\wed"
        mkdir "%install_dir%"
        :: -- Update PATH --
		setx PATH "%install_dir%;%PATH%" >nul && (
			echo Added to PATH: %install_dir%
		) || (
			echo Warning: Could not update PATH permanently
		)
    )
)

:: -- Build and install --
echo Compiling wed@%wed_version%...
go build -ldflags="-s -w -X main.Version=%wed_version%" -o "%install_dir%\wed.exe" ./cmd/wed
if errorlevel 1 (
    echo Error: Build failed
    exit /b 1
)

:: -- Success message --
echo.
echo ✓ Installed to: %install_dir%\wed.exe
echo To verify, run:
echo   wed --version
