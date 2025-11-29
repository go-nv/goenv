@echo off
REM Cross-platform build wrapper for Windows (delegates to Go-based tool)

set TASK=%1
if "%TASK%"=="" set TASK=build

REM Shift arguments and pass the rest to the Go tool
shift
set ARGS=
:parse
if "%1"=="" goto execute
set ARGS=%ARGS% %1
shift
goto parse

:execute
go run scripts/build-tool/main.go -task=%TASK% %ARGS%
