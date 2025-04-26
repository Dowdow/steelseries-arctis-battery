@echo off
echo Checking for rsrc tool...

where rsrc >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo ERROR: rsrc tool not found!
    echo Please install it with: go install github.com/akavel/rsrc@latest
    echo Then add Go bin directory to your PATH if needed
    exit /b 1
)

echo Generating icon resource...

rsrc -ico icon\icon.ico -o rsrc.syso
if %ERRORLEVEL% neq 0 (
    echo ERROR: Failed to generate icon resource.
    exit /b 1
)

echo Building application...
set GOARCH=amd64
set GOOS=windows
set CGO_ENABLED=1

go build -ldflags="-H windowsgui" -o dist\build.exe
if %ERRORLEVEL% neq 0 (
    echo ERROR: Build failed.
    exit /b 1
)

echo Build completed successfully!
