@echo off
:: Simplified Windows installer
setlocal enabledelayedexpansion

:: Binary directory logic (simplified if/else) --
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

:: Install binary
copy /Y wed.exe "%install_dir%\wed.exe"
if errorlevel 1 (
    echo Failed to copy wed.exe to %install_dir%
    exit /b 1
)

:: Success message
echo.
echo Successfully installed wed to:
echo   %install_dir%
echo
echo To verify installation, run:
echo   wed --version
echo.

endlocal
