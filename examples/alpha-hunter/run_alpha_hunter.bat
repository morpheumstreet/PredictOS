@echo off
echo Initializing Alpha Hunter...
echo.

:: Check for Python installation
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Python is not installed or not in your PATH.
    pause
    exit /b
)

:: Install dependencies
echo Installing required libraries (requests, pandas, groq)...
pip install requests pandas groq >nul 2>&1

:: Set PYTHONPATH to current directory
set PYTHONPATH=%PYTHONPATH%;.

:: Run the main script
echo.
python main.py

echo.
echo Cycle finished. Press any key to exit.
pause
