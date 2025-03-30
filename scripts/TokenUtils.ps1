##############################
# PanelBase Token 管理工具腳本
# 用於獲取 JWT 令牌和創建 API 令牌
##############################

# 全局變量
$BaseUrl = "http://localhost:45784"  # 默認 URL，可修改為您的實際 URL
$JwtToken = $null
$ApiToken = $null
$Username = $null
$Password = $null
$TokenExpiration = $null

function Show-Header {
    Write-Host "====================================" -ForegroundColor Cyan
    Write-Host "    PanelBase Token 管理工具        " -ForegroundColor Cyan
    Write-Host "====================================" -ForegroundColor Cyan
    Write-Host
}

function Show-Menu {
    Write-Host "請選擇操作:" -ForegroundColor Yellow
    Write-Host "1. 設置服務器 URL" -ForegroundColor Green
    Write-Host "2. 登錄並獲取 JWT 令牌" -ForegroundColor Green
    Write-Host "3. 創建新的 API 令牌" -ForegroundColor Green
    Write-Host "4. 列出當前令牌" -ForegroundColor Green
    Write-Host "5. 測試令牌有效性" -ForegroundColor Green
    Write-Host "0. 退出" -ForegroundColor Green
    Write-Host
}

function Set-ServerUrl {
    Write-Host "當前服務器 URL: $BaseUrl" -ForegroundColor Magenta
    $newUrl = Read-Host "請輸入新的服務器 URL (直接回車保持當前值)"
    
    if ($newUrl) {
        $script:BaseUrl = $newUrl
        Write-Host "服務器 URL 已更新為: $BaseUrl" -ForegroundColor Green
    }
}

function Get-JwtToken {
    Write-Host "登錄並獲取 JWT 令牌" -ForegroundColor Magenta
    
    $script:Username = Read-Host "用戶名"
    $securePassword = Read-Host "密碼" -AsSecureString
    $bstr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
    $script:Password = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
    
    # 獲取過期時間設置
    $script:TokenExpiration = Read-Host "令牌過期時間 (小時, 直接回車使用默認值 24 小時)"
    if (-not $TokenExpiration) {
        $script:TokenExpiration = "24"
    }
    
    $loginUrl = "$BaseUrl/api/v1/auth/login"
    $body = @{
        username = $Username
        password = $Password
        duration = $TokenExpiration
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri $loginUrl -Method Post -Body $body -ContentType "application/json"
        
        if ($response.status -eq "success") {
            $script:JwtToken = $response.data.token
            Write-Host "登錄成功! JWT 令牌已獲取。" -ForegroundColor Green
            Write-Host "用戶: $($response.data.user.name) ($($response.data.user.username))" -ForegroundColor Green
            Write-Host "角色: $($response.data.user.role)" -ForegroundColor Green
            Write-Host "過期時間: $($response.data.expires) 小時" -ForegroundColor Green
            Write-Host "令牌將在 $(Get-Date).AddHours([int]$response.data.expires) 過期" -ForegroundColor Yellow
            Write-Host
            Write-Host "JWT 令牌:" -ForegroundColor Cyan
            Write-Host $JwtToken -ForegroundColor Gray
            
            return $true
        } else {
            Write-Host "登錄失敗: $($response.message)" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host "請求出錯: $_" -ForegroundColor Red
        return $false
    }
}

function Create-ApiToken {
    if (-not $JwtToken) {
        Write-Host "請先登錄獲取 JWT 令牌" -ForegroundColor Red
        return
    }
    
    Write-Host "創建新的 API 令牌" -ForegroundColor Magenta
    
    $tokenName = Read-Host "令牌名稱"
    $tokenPermissions = Read-Host "權限 (多個權限用逗號分隔, 如: read,write)"
    $tokenPermissionsArray = $tokenPermissions -split ',' | ForEach-Object { $_.Trim() }
    
    $tokenDuration = Read-Host "有效期 (ISO 8601格式, 如: PT1H 表示1小時, PT7D 表示7天)"
    if (-not $tokenDuration) {
        $tokenDuration = "PT1H"  # 默認1小時
    }
    
    $tokenRateLimit = Read-Host "速率限制 (每分鐘請求數, 默認 60)"
    if (-not $tokenRateLimit) {
        $tokenRateLimit = 60
    }
    
    $tokenUrl = "$BaseUrl/api/v1/auth/token"
    $body = @{
        name = $tokenName
        permissions = $tokenPermissionsArray
        duration = $tokenDuration
        rate_limit = [int]$tokenRateLimit
    } | ConvertTo-Json
    
    $headers = @{
        "Authorization" = "Bearer $JwtToken"
        "Content-Type" = "application/json"
    }
    
    try {
        $response = Invoke-RestMethod -Uri $tokenUrl -Method Post -Headers $headers -Body $body
        
        if ($response.status -eq "success") {
            $script:ApiToken = $response.data.token
            Write-Host "API 令牌創建成功!" -ForegroundColor Green
            Write-Host "令牌名稱: $($response.data.name)" -ForegroundColor Green
            Write-Host
            Write-Host "API 令牌:" -ForegroundColor Cyan
            Write-Host $ApiToken -ForegroundColor Gray
        } else {
            Write-Host "創建 API 令牌失敗: $($response.message)" -ForegroundColor Red
        }
    } catch {
        Write-Host "請求出錯: $_" -ForegroundColor Red
        
        # JWT令牌可能已過期，但仍然可以創建API令牌
        if ($_.Exception.Response.StatusCode -eq 401) {
            Write-Host "JWT令牌可能已過期，嘗試重新登錄..." -ForegroundColor Yellow
            if (Get-JwtToken) {
                Write-Host "重新嘗試創建 API 令牌..." -ForegroundColor Yellow
                Create-ApiToken
            }
        }
    }
}

function Show-CurrentTokens {
    Write-Host "當前令牌信息:" -ForegroundColor Magenta
    
    if ($JwtToken) {
        Write-Host "JWT 令牌:" -ForegroundColor Green
        Write-Host $JwtToken -ForegroundColor Gray
    } else {
        Write-Host "未獲取 JWT 令牌" -ForegroundColor Yellow
    }
    
    Write-Host
    
    if ($ApiToken) {
        Write-Host "API 令牌:" -ForegroundColor Green
        Write-Host $ApiToken -ForegroundColor Gray
    } else {
        Write-Host "未創建 API 令牌" -ForegroundColor Yellow
    }
}

function Test-TokenValidity {
    Write-Host "測試令牌有效性" -ForegroundColor Magenta
    
    $tokenType = Read-Host "要測試的令牌類型 (1: JWT, 2: API)"
    $token = $null
    
    switch ($tokenType) {
        "1" { 
            $token = $JwtToken
            if (-not $token) {
                Write-Host "未獲取 JWT 令牌" -ForegroundColor Red
                return
            }
        }
        "2" { 
            $token = $ApiToken
            if (-not $token) {
                Write-Host "未創建 API 令牌" -ForegroundColor Red
                return
            }
        }
        default {
            Write-Host "無效的選擇" -ForegroundColor Red
            return
        }
    }
    
    $testUrl = "$BaseUrl/api/v1/users"
    $headers = @{
        "Authorization" = "Bearer $token"
    }
    
    try {
        $response = Invoke-RestMethod -Uri $testUrl -Method Get -Headers $headers
        Write-Host "令牌有效! 成功訪問資源。" -ForegroundColor Green
        Write-Host "響應狀態: $($response.status)" -ForegroundColor Green
    } catch {
        Write-Host "令牌無效或已過期" -ForegroundColor Red
        Write-Host "錯誤: $_" -ForegroundColor Red
    }
}

function Start-TokenManager {
    Clear-Host
    Show-Header
    
    do {
        Show-Menu
        $choice = Read-Host "請輸入選項"
        
        switch ($choice) {
            "1" { Set-ServerUrl }
            "2" { Get-JwtToken }
            "3" { Create-ApiToken }
            "4" { Show-CurrentTokens }
            "5" { Test-TokenValidity }
            "0" { 
                Write-Host "感謝使用 PanelBase Token 管理工具" -ForegroundColor Cyan
                return 
            }
            default { Write-Host "無效的選項，請重試" -ForegroundColor Red }
        }
        
        Write-Host
        Read-Host "按 Enter 鍵繼續..."
        Clear-Host
        Show-Header
    } while ($true)
}

# 啟動程序
Start-TokenManager 