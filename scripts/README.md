# PanelBase 令牌管理工具

本目錄包含用於管理和解析 PanelBase JWT 和 API 令牌的腳本工具。

## 可用腳本

### 令牌管理工具

1. **TokenUtils.ps1**：Windows PowerShell 互動式令牌管理工具
   - 提供完整的互動式介面
   - 支持設置服務器 URL
   - 支持登錄獲取 JWT 令牌
   - 支持創建 API 令牌
   - 支持測試令牌有效性
   - 適合 Windows 環境使用

2. **TokenUtils.sh**：Linux Shell 互動式令牌管理工具
   - 基於 Bash 的互動式介面
   - 提供與 PowerShell 版本相同的功能
   - 使用 curl 命令進行 API 請求
   - 適合 Linux/macOS 環境使用

### 令牌解析工具

3. **TokenParser.ps1**：Windows PowerShell 令牌解析工具
   - 解析 JWT 和 API 令牌的結構
   - 支持從文件或直接輸入解析令牌
   - 顯示令牌的標頭、載荷和簽名詳細信息
   - 支持將解析結果保存為 JSON 文件

4. **token_parser.sh**：Linux Shell 令牌解析工具
   - 基於 Bash 的令牌解析工具
   - 支持多種 JSON 處理工具 (jq、Python)
   - 提供彩色輸出，便於閱讀
   - 自動檢測令牌過期狀態

## 使用方法

### 令牌管理工具

#### Windows PowerShell 版本

```powershell
.\TokenUtils.ps1
```

#### Linux/macOS Bash 版本

```bash
# 先添加執行權限
chmod +x TokenUtils.sh

# 運行腳本
./TokenUtils.sh
```

運行後會顯示互動式菜單，按照提示進行操作：
1. 設置服務器 URL（默認 http://localhost:45784）
2. 登錄並獲取 JWT 令牌
3. 創建新的 API 令牌
4. 列出當前令牌
5. 測試令牌有效性
6. 保存令牌到文件

### 令牌解析工具

#### Windows PowerShell 版本

```powershell
.\TokenParser.ps1
```

#### Linux/macOS Bash 版本

```bash
# 先添加執行權限
chmod +x token_parser.sh

# 運行腳本
./token_parser.sh
```

運行後會顯示輸入選項：
1. 直接輸入令牌
2. 從文件讀取令牌
3. 退出

## 功能說明

### 令牌管理功能

- **獲取 JWT 令牌**：通過用戶名和密碼登錄系統，獲取 JWT 令牌。可以設置令牌的過期時間（小時）。
- **創建 API 令牌**：使用 JWT 令牌創建長期有效的 API 令牌，可以設置名稱、權限、有效期和速率限制。
- **測試令牌有效性**：測試 JWT 令牌或 API 令牌是否有效，通過訪問系統 API 檢查認證狀態。
- **保存令牌到文件**：將獲取的令牌保存到本地文件中，方便後續使用。

### 令牌解析功能

- **解析令牌結構**：分解 JWT 令牌的三個部分（標頭、載荷、簽名）
- **Base64URL 解碼**：解碼令牌的標頭和載荷部分
- **顯示關鍵信息**：提取並顯示令牌中的重要字段（用戶ID、過期時間等）
- **檢查令牌狀態**：自動計算令牌是否已過期
- **保存詳細信息**：將解析結果保存為 JSON 文件，便於進一步分析

## 注意事項

1. PowerShell 腳本需要在 PowerShell 5.0 或更高版本中運行
2. Bash 腳本需要安裝 curl 和 base64 命令
3. 令牌解析工具建議安裝 jq 或 Python 以獲得更好的 JSON 格式化體驗
4. API 令牌使用 ISO 8601 格式指定過期時間：
   - PT1H = 1小時
   - PT24H = 24小時
   - P7D = 7天
   - P30D = 30天
5. 令牌使用 Bearer 認證方式，在 HTTP 請求中使用 Authorization 頭
6. 如果 JWT 令牌過期，腳本會提示您重新登錄 