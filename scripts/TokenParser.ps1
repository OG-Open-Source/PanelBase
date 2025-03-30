#!/usr/bin/env pwsh
##############################
# PanelBase Token 解析工具 (PowerShell版)
# 用於解析 JWT 令牌和 API 令牌的結構
##############################

function Show-Banner {
    Write-Host "====================================" -ForegroundColor Cyan
    Write-Host "    PanelBase Token 解析工具        " -ForegroundColor Cyan
    Write-Host "====================================" -ForegroundColor Cyan
    Write-Host
}

function Decode-Base64Url {
    param (
        [Parameter(Mandatory=$true)]
        [string]$Base64Url
    )
    
    # 替換URL安全字符為標準Base64字符
    $Base64 = $Base64Url.Replace('-', '+').Replace('_', '/')
    
    # 添加填充
    switch ($Base64.Length % 4) {
        0 { break }
        2 { $Base64 += "==" }
        3 { $Base64 += "=" }
    }
    
    try {
        $Bytes = [System.Convert]::FromBase64String($Base64)
        $Text = [System.Text.Encoding]::UTF8.GetString($Bytes)
        return $Text
    } catch {
        Write-Host "Base64URL解碼錯誤: $_" -ForegroundColor Red
        return $null
    }
}

function Parse-JwtToken {
    param (
        [Parameter(Mandatory=$true)]
        [string]$TokenString
    )
    
    # 分割令牌
    $TokenParts = $TokenString.Split('.')
    
    if ($TokenParts.Count -ne 3) {
        Write-Host "錯誤: 無效的JWT格式。JWT應該包含3部分 (標頭.載荷.簽名)" -ForegroundColor Red
        return
    }
    
    # 解碼標頭
    $HeaderJson = Decode-Base64Url -Base64Url $TokenParts[0]
    $Header = $HeaderJson | ConvertFrom-Json
    
    # 解碼載荷
    $PayloadJson = Decode-Base64Url -Base64Url $TokenParts[1]
    $Payload = $PayloadJson | ConvertFrom-Json
    
    # 簽名（保持原樣，因為它是二進制數據）
    $Signature = $TokenParts[2]
    
    # 解析時間戳
    if ($Payload.exp) {
        $ExpirationDate = (Get-Date "1970-01-01").AddSeconds($Payload.exp)
        $ExpirationDateLocal = $ExpirationDate.ToLocalTime()
    } else {
        $ExpirationDateLocal = "未設置"
    }
    
    if ($Payload.iat) {
        $IssuedDate = (Get-Date "1970-01-01").AddSeconds($Payload.iat)
        $IssuedDateLocal = $IssuedDate.ToLocalTime()
    } else {
        $IssuedDateLocal = "未設置"
    }
    
    # 顯示令牌信息
    Write-Host "===== JWT令牌解析結果 =====" -ForegroundColor Yellow
    
    Write-Host "`n[標頭 (Header)]" -ForegroundColor Green
    Write-Host "原始數據: $($TokenParts[0])" -ForegroundColor Gray
    Write-Host "解碼後:"
    $Header | Format-List | Out-String | Write-Host
    
    Write-Host "[載荷 (Payload)]" -ForegroundColor Green
    Write-Host "原始數據: $($TokenParts[1])" -ForegroundColor Gray
    Write-Host "解碼後:"
    $Payload | Format-List | Out-String | Write-Host
    
    Write-Host "[簽名 (Signature)]" -ForegroundColor Green
    Write-Host "$Signature" -ForegroundColor Gray
    
    Write-Host "`n[關鍵信息]" -ForegroundColor Magenta
    
    if ($Payload.type) {
        Write-Host "令牌類型: $($Payload.type)" -ForegroundColor Cyan
    }
    
    if ($Payload.user_id) {
        Write-Host "用戶ID: $($Payload.user_id)" -ForegroundColor Cyan
    }
    
    if ($Payload.sub) {
        Write-Host "主題 (Subject): $($Payload.sub)" -ForegroundColor Cyan
    }
    
    if ($Payload.username) {
        Write-Host "用戶名: $($Payload.username)" -ForegroundColor Cyan
    }
    
    if ($Payload.role) {
        Write-Host "角色: $($Payload.role)" -ForegroundColor Cyan
    }
    
    if ($Payload.exp) {
        Write-Host "過期時間: $ExpirationDateLocal" -ForegroundColor Cyan
        
        # 檢查令牌是否已過期
        $Now = Get-Date
        if ($ExpirationDate -lt $Now) {
            Write-Host "狀態: 已過期" -ForegroundColor Red
        } else {
            $TimeRemaining = $ExpirationDate - $Now
            Write-Host "狀態: 有效 (剩餘 $([math]::Round($TimeRemaining.TotalHours, 2)) 小時)" -ForegroundColor Green
        }
    }
    
    if ($Payload.iat) {
        Write-Host "簽發時間: $IssuedDateLocal" -ForegroundColor Cyan
    }
    
    if ($Payload.jti) {
        Write-Host "JWT ID: $($Payload.jti)" -ForegroundColor Cyan
    }
    
    if ($Payload.token_id) {
        Write-Host "令牌ID: $($Payload.token_id)" -ForegroundColor Cyan
    }
    
    # 顯示其他自定義字段
    $StandardClaims = @("exp", "iat", "nbf", "jti", "iss", "aud", "sub", "type", "user_id", "username", "role", "token_id")
    $CustomClaims = $Payload.PSObject.Properties | Where-Object { $_.Name -notin $StandardClaims }
    
    if ($CustomClaims) {
        Write-Host "`n[自定義字段]" -ForegroundColor Yellow
        foreach ($Claim in $CustomClaims) {
            Write-Host "$($Claim.Name): $($Claim.Value)" -ForegroundColor White
        }
    }
}

function Save-ParsedToken {
    param (
        [Parameter(Mandatory=$true)]
        [string]$TokenString,
        [string]$OutputFile = "token_details.json"
    )
    
    # 分割令牌
    $TokenParts = $TokenString.Split('.')
    
    if ($TokenParts.Count -ne 3) {
        Write-Host "錯誤: 無效的JWT格式" -ForegroundColor Red
        return
    }
    
    # 解碼標頭
    $HeaderJson = Decode-Base64Url -Base64Url $TokenParts[0]
    $Header = $HeaderJson | ConvertFrom-Json
    
    # 解碼載荷
    $PayloadJson = Decode-Base64Url -Base64Url $TokenParts[1]
    $Payload = $PayloadJson | ConvertFrom-Json
    
    # 創建結果對象
    $Result = [PSCustomObject]@{
        token = $TokenString
        header = $Header
        payload = $Payload
        signature = $TokenParts[2]
    }
    
    # 保存到文件
    $Result | ConvertTo-Json -Depth 10 | Out-File -FilePath $OutputFile -Encoding utf8
    
    Write-Host "令牌詳細信息已保存到: $OutputFile" -ForegroundColor Green
}

function Main {
    Show-Banner
    
    # 提示用戶輸入令牌或從文件讀取
    Write-Host "請選擇輸入方式:" -ForegroundColor Yellow
    Write-Host "1. 直接輸入令牌" -ForegroundColor Green
    Write-Host "2. 從文件讀取令牌" -ForegroundColor Green
    Write-Host "3. 退出" -ForegroundColor Green
    
    $choice = Read-Host "請選擇"
    
    switch ($choice) {
        "1" {
            $token = Read-Host "請輸入JWT令牌或API令牌"
            
            if (-not [string]::IsNullOrWhiteSpace($token)) {
                Parse-JwtToken -TokenString $token
                
                $saveChoice = Read-Host "是否保存解析結果到文件? (y/n)"
                if ($saveChoice -eq "y") {
                    $fileName = Read-Host "輸入文件名 (默認: token_details.json)"
                    if ([string]::IsNullOrWhiteSpace($fileName)) {
                        $fileName = "token_details.json"
                    }
                    Save-ParsedToken -TokenString $token -OutputFile $fileName
                }
            } else {
                Write-Host "錯誤: 未提供令牌" -ForegroundColor Red
            }
        }
        "2" {
            $filePath = Read-Host "請輸入令牌文件路徑"
            
            if (Test-Path $filePath) {
                $fileContent = Get-Content -Path $filePath -Raw
                
                # 嘗試提取令牌（移除可能的註釋和空行）
                $tokenLines = $fileContent -split "`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) -and -not $_.StartsWith("#") }
                
                if ($tokenLines.Count -gt 1) {
                    Write-Host "文件包含多行，請選擇要解析的令牌:" -ForegroundColor Yellow
                    
                    for ($i = 0; $i -lt $tokenLines.Count; $i++) {
                        Write-Host "$($i+1). $($tokenLines[$i].Substring(0, [Math]::Min(30, $tokenLines[$i].Length)))..." -ForegroundColor Green
                    }
                    
                    $lineChoice = Read-Host "請選擇 (1-$($tokenLines.Count))"
                    $index = [int]$lineChoice - 1
                    
                    if ($index -ge 0 -and $index -lt $tokenLines.Count) {
                        $token = $tokenLines[$index].Trim()
                        Parse-JwtToken -TokenString $token
                        
                        $saveChoice = Read-Host "是否保存解析結果到文件? (y/n)"
                        if ($saveChoice -eq "y") {
                            $fileName = Read-Host "輸入文件名 (默認: token_details.json)"
                            if ([string]::IsNullOrWhiteSpace($fileName)) {
                                $fileName = "token_details.json"
                            }
                            Save-ParsedToken -TokenString $token -OutputFile $fileName
                        }
                    } else {
                        Write-Host "無效的選擇" -ForegroundColor Red
                    }
                } else {
                    $token = $tokenLines[0].Trim()
                    Parse-JwtToken -TokenString $token
                    
                    $saveChoice = Read-Host "是否保存解析結果到文件? (y/n)"
                    if ($saveChoice -eq "y") {
                        $fileName = Read-Host "輸入文件名 (默認: token_details.json)"
                        if ([string]::IsNullOrWhiteSpace($fileName)) {
                            $fileName = "token_details.json"
                        }
                        Save-ParsedToken -TokenString $token -OutputFile $fileName
                    }
                }
            } else {
                Write-Host "錯誤: 文件不存在" -ForegroundColor Red
            }
        }
        "3" {
            Write-Host "退出程序" -ForegroundColor Cyan
            return
        }
        default {
            Write-Host "無效選擇，請重試" -ForegroundColor Red
        }
    }
    
    Write-Host "`n感謝使用 PanelBase Token 解析工具!" -ForegroundColor Cyan
}

# 執行主程序
Main 