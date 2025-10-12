@echo off
REM Build script for goenv on Windows (Batch)
REM For more advanced features, use build.ps1 (PowerShell)

setlocal enabledelayedexpansion

REM Build variables
set BINARY_NAME=goenv.exe

REM Get version
if exist APP_VERSION (
    set /p VERSION=<APP_VERSION
) else (
    set VERSION=dev
)

REM Get commit SHA
for /f "delims=" %%i in ('git rev-parse --short HEAD 2^>nul') do set COMMIT_SHA=%%i
if "!COMMIT_SHA!"=="" set COMMIT_SHA=unknown

REM Get build time (simplified for batch)
for /f "tokens=*" %%i in ('powershell -Command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ' -AsUTC"') do set BUILD_TIME=%%i

set LDFLAGS=-ldflags "-X main.version=!VERSION! -X main.commit=!COMMIT_SHA! -X main.buildTime=!BUILD_TIME!"

REM Check for command argument
if "%1"=="" goto build
if /i "%1"=="build" goto build
if /i "%1"=="test" goto test
if /i "%1"=="test-windows" goto test-windows
if /i "%1"=="clean" goto clean
if /i "%1"=="dev-deps" goto dev-deps
if /i "%1"=="generate-embedded" goto generate-embedded
if /i "%1"=="version" goto version
if /i "%1"=="help" goto help
if /i "%1"=="/?" goto help
if /i "%1"=="-h" goto help
goto help

:build
    echo Building %BINARY_NAME%...
    go build %LDFLAGS% -o %BINARY_NAME% .
    if %ERRORLEVEL% NEQ 0 (
        echo Build failed!
        exit /b 1
    )
    echo Build successful: %BINARY_NAME%

    REM Create bin directory for backward compatibility
    if not exist bin mkdir bin
    copy /Y %BINARY_NAME% bin\goenv.exe >nul
    echo Copied to bin\goenv.exe for backward compatibility
    goto end

:test
    echo Running tests...
    go test -v ./...
    if %ERRORLEVEL% NEQ 0 (
        echo Tests failed!
        exit /b 1
    )
    echo Tests passed!
    goto end

:test-windows
    echo Testing Windows compatibility...
    go run scripts/test_windows_compatibility/main.go
    if %ERRORLEVEL% NEQ 0 (
        echo Windows compatibility test failed!
        exit /b 1
    )
    echo Windows compatibility verified!
    goto end

:clean
    echo Cleaning build artifacts...
    if exist %BINARY_NAME% del /F %BINARY_NAME%
    if exist bin rmdir /S /Q bin
    if exist dist rmdir /S /Q dist
    go clean
    echo Clean complete!
    goto end

:dev-deps
    echo Downloading Go module dependencies...
    go mod download
    echo Tidying Go modules...
    go mod tidy
    echo Dependencies updated!
    goto end

:generate-embedded
    echo Generating embedded versions from go.dev API...
    go run scripts/generate_embedded_versions/main.go
    if %ERRORLEVEL% NEQ 0 (
        echo Failed to generate embedded versions!
        exit /b 1
    )
    echo Embedded versions generated successfully!
    goto end

:version
    echo goenv Build Information
    echo   Version:    !VERSION!
    echo   Commit:     !COMMIT_SHA!
    echo   Build Time: !BUILD_TIME!
    goto end

:help
    echo.
    echo goenv Build Script for Windows (Batch)
    echo.
    echo Usage: build.bat [TASK]
    echo.
    echo Tasks:
    echo   build              Build the goenv binary (default)
    echo   test               Run all tests
    echo   test-windows       Test Windows compatibility
    echo   clean              Remove built binaries and clean build artifacts
    echo   dev-deps           Download and tidy Go module dependencies
    echo   generate-embedded  Generate embedded versions from go.dev API
    echo   version            Show version information
    echo   help               Show this help message
    echo.
    echo For advanced features (install, cross-build), use build.ps1 (PowerShell):
    echo   powershell -ExecutionPolicy Bypass -File build.ps1 [TASK]
    echo.
    echo Examples:
    echo   build.bat build
    echo   build.bat test
    echo   build.bat clean
    echo.
    goto end

:end
endlocal
