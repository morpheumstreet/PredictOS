@echo off
echo Initializing Alpha Hunter (DEMO MODE)...
echo.

:: Check for Python installation
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Python is not installed or not in your PATH.
    pause
    exit /b
)

:: Run the demo script
echo.
python run_demo.py

:: Automatically pop out the results file
start logs\latest_results.json

echo.
echo Cycle finished. Press any key to exit.
pause
