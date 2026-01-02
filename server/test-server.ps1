# Quick Server Test Script
# Make sure the server is running before executing this script

Write-Host "=== Testing Server Endpoints ===" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:3000"

try {
    # 1. Health Check
    Write-Host "1. Testing Health Check..." -ForegroundColor Yellow
    $health = Invoke-WebRequest -Uri "$baseUrl/" -UseBasicParsing
    if ($health.Content -like "*Simulation Orchestration Server*") {
        Write-Host "   ✓ Health check passed" -ForegroundColor Green
    } else {
        Write-Host "   ✗ Health check failed" -ForegroundColor Red
    }

    # 2. Get Simulations
    Write-Host "2. Testing Get Simulations..." -ForegroundColor Yellow
    $sims = Invoke-WebRequest -Uri "$baseUrl/api/simulations" -UseBasicParsing | ConvertFrom-Json
    Write-Host "   ✓ Found $($sims.Count) simulation(s)" -ForegroundColor Green

    # 3. Get Logs
    Write-Host "3. Testing Get Logs..." -ForegroundColor Yellow
    $logs = Invoke-WebRequest -Uri "$baseUrl/api/logs" -UseBasicParsing | ConvertFrom-Json
    Write-Host "   ✓ Found $($logs.Count) log entry/entries" -ForegroundColor Green

    # 4. Get Scenario
    Write-Host "4. Testing Get Scenario..." -ForegroundColor Yellow
    try {
        $scenario = Invoke-WebRequest -Uri "$baseUrl/api/scenario" -UseBasicParsing | ConvertFrom-Json
        Write-Host "   ✓ Current scenario: $($scenario.name) with $($scenario.rules) rules" -ForegroundColor Green
    } catch {
        Write-Host "   ⚠ No scenario loaded (this is OK if you haven't loaded one)" -ForegroundColor Yellow
    }

    # 5. Get Stored Scenarios
    Write-Host "5. Testing Get Stored Scenarios..." -ForegroundColor Yellow
    $stored = Invoke-WebRequest -Uri "$baseUrl/api/scenarios" -UseBasicParsing | ConvertFrom-Json
    Write-Host "   ✓ Found $($stored.Count) stored scenario(s)" -ForegroundColor Green

    Write-Host ""
    Write-Host "=== All Tests Passed! ===" -ForegroundColor Green

} catch {
    Write-Host ""
    Write-Host "=== Error ===" -ForegroundColor Red
    Write-Host "Make sure the server is running on port 3000" -ForegroundColor Red
    Write-Host "Start it with: .\server.exe -port 3000" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
}
