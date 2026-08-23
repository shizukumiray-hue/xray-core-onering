@echo off
REM Quick start script for Xray-Core Onering Multi-CDN (Windows)

echo ==========================================
echo Xray-Core Onering Multi-CDN Quick Start
echo ==========================================
echo.

REM Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set BINARY_NAME=xray-windows-64.exe
) else if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set BINARY_NAME=xray-windows-arm64-v8a.exe
) else (
    set BINARY_NAME=xray-windows-32.exe
)

echo Detected: Windows %PROCESSOR_ARCHITECTURE%
echo Binary: %BINARY_NAME%
echo.

REM Check if binary exists
if exist "xray.exe" (
    echo [OK] Binary found: xray.exe
    set XRAY_BIN=xray.exe
) else if exist "%BINARY_NAME%" (
    echo [OK] Binary found: %BINARY_NAME%
    set XRAY_BIN=%BINARY_NAME%
) else (
    echo [ERROR] Binary not found!
    echo.
    echo Please download from: https://github.com/YOUR_USERNAME/xray-core-onering/releases
    echo Or build: go build -o xray.exe ./main
    pause
    exit /b 1
)

REM Check if config exists
if not exist "config.json" (
    echo.
    echo Creating sample Multi-CDN config...
    
    (
        echo {
        echo   "log": {
        echo     "loglevel": "warning"
        echo   },
        echo   "inbounds": [{
        echo     "port": 10808,
        echo     "protocol": "socks",
        echo     "settings": {
        echo       "udp": true
        echo     }
        echo   }],
        echo   "outbounds": [{
        echo     "protocol": "vmess",
        echo     "settings": {
        echo       "vnext": [{
        echo         "address": "your-server.com",
        echo         "port": 443,
        echo         "users": [{"id": "YOUR-UUID-HERE"}]
        echo       }]
        echo     },
        echo     "streamSettings": {
        echo       "network": "ws",
        echo       "security": "tls",
        echo       "tlsSettings": {
        echo         "serverName": "onering-multi:your-server.com",
        echo         "fingerprint": "chrome",
        echo         "multiCDN": {
        echo           "enabled": true,
        echo           "strategy": "health-based",
        echo           "providers": [
        echo             {
        echo               "name": "cloudflare",
        echo               "bugDomain": "zoom.us",
        echo               "priority": 100,
        echo               "isps": ["telkomsel"]
        echo             },
        echo             {
        echo               "name": "cloudfront",
        echo               "bugDomain": "teams.microsoft.com",
        echo               "priority": 90,
        echo               "isps": ["xl", "indosat"]
        echo             }
        echo           ],
        echo           "healthCheck": {
        echo             "enabled": true,
        echo             "interval": "30s"
        echo           },
        echo           "evasion": {
        echo             "enableRotation": true,
        echo             "rotateInterval": "5m"
        echo           }
        echo         }
        echo       }
        echo     }
        echo   }]
        echo }
    ) > config.json
    
    echo [OK] Sample config created: config.json
    echo.
    echo WARNING: Edit config.json and replace:
    echo    - your-server.com with your actual server
    echo    - YOUR-UUID-HERE with your UUID
    echo.
    pause
)

REM Validate config
echo.
echo Validating configuration...
%XRAY_BIN% test -config config.json
if %ERRORLEVEL% EQU 0 (
    echo [OK] Configuration is valid
) else (
    echo [ERROR] Configuration has errors. Please fix config.json
    pause
    exit /b 1
)

REM Show Multi-CDN info
echo.
echo ==========================================
echo Multi-CDN Features Enabled
echo ==========================================
echo.
echo [OK] Multi-CDN Anti-DPI bypass
echo [OK] Automatic failover
echo [OK] Health monitoring
echo [OK] ISP-specific optimization
echo [OK] DPI evasion techniques
echo.

REM Start Xray
echo Starting Xray-Core Onering Multi-CDN...
echo.
echo SOCKS5 Proxy: 127.0.0.1:10808
echo.
echo Press Ctrl+C to stop
echo.

%XRAY_BIN% run -config config.json

pause
