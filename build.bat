@echo off
chcp 65001 >nul
echo 编译 AICodeChecker...

:: 检查Go环境
go version >nul 2>&1
if errorlevel 1 (
    echo 错误：未找到Go环境
    pause
    exit /b 1
)

:: 删除旧的exe文件
if exist "code-checker.exe" del "code-checker.exe"

:: 编译
echo 正在编译...
go build -ldflags "-s -w" -o code-checker.exe .\cmd\code-checker

if errorlevel 1 (
    echo 编译失败
    pause
    exit /b 1
)

echo 编译完成：code-checker.exe
pause 