# PanelBase

輕量級 Linux 服務器運維管理面板，採用 Go 語言開發，具備高效安全的特性。

## 特點

- 輕量級：佔用系統資源極少，適合任何規格的服務器
- 高安全性：採用最新的安全實踐和密碼學技術
- 插件系統：支持擴展功能的插件架構
- 主題系統：可自訂界面外觀
- 多語言支持：支持中文、英文等多種語言
- 全面監控：系統資源、服務狀態實時監控
- 便捷管理：網站、數據庫、容器等一站式管理

## 安裝

### 快速安裝

```bash
curl -sSL https://example.com/install.sh | bash
```

### 手動安裝

1. 下載最新版本

```bash
wget https://github.com/OG-Open-Source/PanelBase/releases/latest/download/panelbase.tar.gz
```

2. 解壓安裝

```bash
tar -zxvf panelbase.tar.gz
cd panelbase
./install.sh
```

## 使用

安裝完成後，訪問 `http://your-server-ip:8080` 進入管理面板。

初始登入資訊：

- 用戶名：admin
- 密碼：admin

## 配置指南

詳細的配置說明請參閱以下文檔：

- [配置指南](docs/CONFIG_GUIDE.md)：詳細說明主題、插件和命令的配置方法
- [詳細信息](docs/INFO.md)：主題與插件格式規範

## API 使用

PanelBase 提供了完整的 API 接口，支持以下操作：

### 主題相關 API

- **GET /api/v1/themes**: 獲取所有主題或特定主題信息

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" http://your-server-ip:8080/api/v1/themes
  curl -H "Authorization: Bearer YOUR_TOKEN" -X GET -d '{"id":"theme_id"}' http://your-server-ip:8080/api/v1/themes
  ```

- **POST /api/v1/themes**: 安裝新主題或切換當前主題

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"url":"theme_source_url"}' http://your-server-ip:8080/api/v1/themes
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"id":"theme_id","status":"switch"}' http://your-server-ip:8080/api/v1/themes
  ```

- **PUT /api/v1/themes**: 更新所有已安裝主題

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PUT http://your-server-ip:8080/api/v1/themes
  ```

- **PATCH /api/v1/themes**: 更新特定主題

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PATCH -d '{"id":"theme_id"}' http://your-server-ip:8080/api/v1/themes
  ```

- **DELETE /api/v1/themes**: 删除所有非當前主題或特定主題
  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE http://your-server-ip:8080/api/v1/themes
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE -d '{"id":"theme_id"}' http://your-server-ip:8080/api/v1/themes
  ```

### 插件相關 API

- **GET /api/v1/plugins**: 獲取所有插件或特定插件信息

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" http://your-server-ip:8080/api/v1/plugins
  curl -H "Authorization: Bearer YOUR_TOKEN" -X GET -d '{"id":"plugin_id"}' http://your-server-ip:8080/api/v1/plugins
  ```

- **POST /api/v1/plugins**: 安裝新插件

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"url":"plugin_source_url"}' http://your-server-ip:8080/api/v1/plugins
  ```

- **PUT /api/v1/plugins**: 更新所有已安裝插件

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PUT http://your-server-ip:8080/api/v1/plugins
  ```

- **PATCH /api/v1/plugins**: 更新特定插件

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PATCH -d '{"id":"plugin_id"}' http://your-server-ip:8080/api/v1/plugins
  ```

- **DELETE /api/v1/plugins**: 删除所有插件或特定插件
  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE http://your-server-ip:8080/api/v1/plugins
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE -d '{"id":"plugin_id"}' http://your-server-ip:8080/api/v1/plugins
  ```

### 命令相關 API

- **GET /api/v1/commands**: 獲取所有命令或特定命令信息

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" http://your-server-ip:8080/api/v1/commands
  curl -H "Authorization: Bearer YOUR_TOKEN" -X GET -d '{"id":"command_id"}' http://your-server-ip:8080/api/v1/commands
  ```

- **POST /api/v1/commands**: 安裝新命令

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"url":"command_source_url"}' http://your-server-ip:8080/api/v1/commands
  ```

- **PUT /api/v1/commands**: 更新所有已安裝命令

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PUT http://your-server-ip:8080/api/v1/commands
  ```

- **PATCH /api/v1/commands**: 更新特定命令

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PATCH -d '{"id":"command_id"}' http://your-server-ip:8080/api/v1/commands
  ```

- **DELETE /api/v1/commands**: 删除所有命令或特定命令
  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE http://your-server-ip:8080/api/v1/commands
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE -d '{"id":"command_id"}' http://your-server-ip:8080/api/v1/commands
  ```

## 更新日誌

### 2024 年 7 月 14 日

- 改進代碼性能和體驗
  - 移除所有與持續時間解析和API令牌創建相關的調試日誌信息
  - 優化核心功能代碼，減少屏幕輸出，提升執行效率
  - 保留重要邏輯功能，僅移除開發階段的調試信息
  - 改善系統輸出的清潔度，以提供更專業的使用體驗

### 2024 年 7 月 13 日

- 优化API令牌持续时间处理
  - 移除API令牌最小和最大持续时间限制，允许用户完全自定义过期时间
  - 简化持续时间解析，仅支持ISO 8601标准格式（如`P3Y6M4DT12H30M5S`、`PT1H`）
  - 移除多级持续时间格式解析，确保系统一致性和标准化
  - 完善日志输出，提供更清晰的持续时间解析过程信息
  - 统一所有与令牌相关函数的持续时间处理方式，使用相同的逻辑

- 增强ISO 8601持续时间格式支持
  - 改进对复杂ISO 8601持续时间格式的解析，如`P3Y6M4DT12H30M5S`
  - 添加多级持续时间格式解析逻辑，支持ISO 8601、Go标准格式和数字小时数
  - 扩展调试日志，显示持续时间的解析过程和结果
  - 统一所有持续时间的最小和最大限制（1小时至720小时）
  - 优化令牌创建过程的持续时间显示，更直观地展示小时和分钟
  - 确保所有API令牌创建方法使用相同的持续时间解析逻辑

- 修复API token生成和验证问题
  - 修正API token的签名密钥传递问题，确保使用正确的JWT密钥
  - 移除`generateAPITokenJWT`函数中的硬编码密钥，改为使用配置中的密钥
  - 确保所有API token的生成都使用相同的密钥进行签名
  - 优化调试日志，便于追踪令牌创建和验证过程

- 修复API token过期时间计算问题
  - 修正API token创建时过期时间计算错误，确保正确应用持续时间
  - 添加调试日志，以便于排查时间计算问题
  - 统一使用单一时间变量，避免多次调用time.Now()导致的微小差异
  - 确保令牌过期时间(exp)正确设置为当前时间加上持续时间

- 修正时间字段序列化问题
  - 添加自定义 `JsonTime` 类型处理时间字段的零值
  - 确保 `last_login`、`last_used` 和 `last_reset_time` 等时间字段在未设置时序列化为 `null`
  - 统一系统中所有时间字段的处理方式
  - 解决 users.json 中时间字段显示为 "0001-01-01" 的问题
  - 修复配置初始化和用户配置加载中的时间字段类型转换问题
  - 确保所有地方使用 `JsonTime` 类型处理时间值
  - 修复 API token 创建时的 `last_used` 和 `last_reset_time` 初始化为零值，正确显示为 `null`
  - 统一时间字段处理逻辑，确保首次使用前所有时间都显示为 `null`
  - 添加 API token 使用统计追踪功能，自动更新 `last_used` 和 `usage_count` 字段
  - 优化认证中间件，通过匹配 token 字符串查找和更新对应的 API token 记录

### 2024 年 7 月 12 日

- 添加令牌解析工具
  - 创建 PowerShell 版本 TokenParser.ps1 脚本，用于解析 JWT 和 API 令牌
  - 创建 Bash 版本 token_parser.sh 脚本，提供相同的功能
  - 支持令牌结构分析，显示标头、载荷和签名部分
  - 自动检测令牌过期状态并显示剩余有效时间
  - 支持从文件读取或直接输入令牌
  - 可将解析结果保存为 JSON 文件
  - 添加彩色输出，提升使用体验
  - 自动识别令牌中的关键字段，包括 type、username、user_id 等
  - 对令牌解析工具进行兼容性优化，支持不同环境
  - 修正令牌中 username 和 user_id 字段的解析逻辑
  - 更新令牌类型的识别机制，区分 JWT 和 API 令牌

### 2024 年 7 月 9 日

- 添加配置文件自動檢查和生成功能
  - 啟動時自動檢測 `/configs/` 目錄及必要的配置文件是否存在
  - 當缺少配置目錄或文件時，自動創建相應的默認配置
  - 添加 `commands.json` 和 `plugins.json` 空文件自動生成功能
  - 添加 `config.yaml` 自動配置功能，從註冊端口(1024-49151)隨機選擇可用端口
  - 添加 `themes.json` 自動配置功能，使用默認主題設置
  - 添加 `users.json` 自動生成功能，創建默認 admin 用戶並加密密碼
  - 新增隨機值生成工具包，支持生成用戶ID、JWT密鑰、API令牌等
  - 新增密碼加密工具包，支持使用 bcrypt 對密碼進行加密和驗證
  - 優化端口選擇範圍，使用標準的註冊端口範圍(1024-49151)
  - 增加端口有效性檢查，確保選擇的端口在有效範圍內
  - 解決了首次安裝時需要手動創建配置文件的問題
  - 簡化了部署流程，降低了安裝門檻
  - 自動創建基本目錄結構，包括 `/commands/`、`/plugins/` 和 `/web/` 目錄
- 重構用戶配置數據結構
  - 移除 User 結構中的冗餘 ID 欄位
  - 使用 map 的 key 作為用戶唯一識別符
  - 生成隨機的用戶 ID 而不是使用固定的 "admin"
  - 更新相關處理邏輯以適應新的數據結構
  - 添加工具函數簡化用戶 ID 查找
  - 修復處理器中使用 User.ID 欄位的代碼
  - 更新 JWT 令牌生成機制，使用 map key 作為用戶 ID
  - 保持向後兼容性，確保舊配置仍然可用
- 系統日誌英文化
  - 將所有日誌輸出更新為英文，提高國際化能力
  - 包含啟動訊息、配置文件檢測、警告和錯誤訊息
  - 保持錯誤訊息清晰明確，方便故障排除
  - 保證配置文件結構和格式不變，只更新日誌語言
- API 響應格式標準化
  - 統一所有 API 響應格式，採用 `status`、`message` 和 `data` 三層結構
  - 為錯誤響應添加詳細說明和相關數據
  - 實現用戶友好的錯誤訊息，方便客戶端處理
  - 保持 API 響應一致性，提高開發體驗
- 修復用戶登入功能
  - 修正 GetUser 方法，支持通過用戶名而非 ID 查找用戶
  - 確保用戶登入 API 能夠正確識別存在的用戶
  - 更新認證中間件錯誤處理邏輯，確保安全性和友好性
  - 解決了用戶無法登入的問題
- 優化 API 路徑和功能
  - 更新登入路徑為 `/api/v1/auth/login`，符合 RESTful 規範
  - 添加 API 令牌創建端點 `/api/v1/auth/token`
  - 實現 UsersConfig.Save 方法，確保用戶數據保存功能正常工作
  - 修復最後登入時間 (last_login) 無法正確記錄問題
  - 標準化所有中間件的錯誤響應格式
  - 完善 API 令牌創建功能，支持權限控制和有效期設置
- 改進用戶登入時間處理
  - 新用戶首次登入前，`last_login` 值顯示為 `null`
  - 登入後正確更新並顯示 `last_login` 時間
  - 使用 ISO 8601 格式（RFC3339）標準化時間顯示
  - 添加時間格式化工具函數，統一處理零值時間
- 優化 Token 驗證機制
  - 重構 JWT 令牌驗證邏輯，增強其健壯性
  - 簡化令牌類型檢測，支持自動識別 JWT 和 API 令牌
  - 修復認證中間件對 JWT 令牌的處理邏輯
  - 解決「未知 token 類型」的錯誤問題
  - 明確在令牌 claims 中添加類型標識
  - 改進 API 令牌創建和驗證流程
  - 優化令牌驗證失敗時的錯誤提示
- 改進 API 令牌創建功能
  - 修復已過期 JWT 令牌無法創建 API 令牌的問題
  - 針對 API 令牌創建請求特殊處理，允許已過期但簽名有效的 JWT 令牌
  - 添加 `CreateAPITokenWithSecret` 方法，使用指定的 JWT 密鑰創建 API 令牌
  - 確保 API 令牌使用與系統相同的 JWT 密鑰進行簽名
  - 優化錯誤處理和響應格式
- 增強 JWT 令牌安全機制
  - 為 JWT 令牌添加過期時間限制，最長 7 天
  - 為 API 令牌添加過期時間限制，最長 30 天
  - 優化過期時間解析邏輯，支持多種時間格式
  - 為 JWT 令牌和 API 令牌添加默認過期時間
  - 實現過期時間上限控制，防止永久有效令牌帶來的安全風險
  - 確保 API 令牌至少有 1 小時的有效期
- 優化 API 令牌數據結構
  - 將 User.API 從數組類型重構為映射（map）類型
  - 使用令牌ID作為映射的鍵（key），提高查詢效率
  - 從 APIToken 結構中移除冗餘的 ID 欄位
  - 優化令牌管理相關方法，包括創建、更新、刪除和重置
  - 改進令牌數據的序列化和反序列化處理
  - 確保向後兼容性，支持舊版本配置自動遷移
  - 減少數據存儲冗餘，提高系統性能

### 2024 年 7 月 8 日

- 統一 API 回應格式規範
  - 設定所有 API 回應使用統一的 JSON 格式：`{status, message, data}`
  - `status` 欄位為 "success" 或 "failure"
  - `message` 欄位包含操作結果的描述信息
  - `data` 欄位（可選）包含返回的數據內容
  - 更新 PluginAPIResponse 結構確保一致性
  - 添加詳細的 API 回應格式規範文檔

### 2024 年 7 月 7 日

- 添加詳細配置指南文檔
  - 創建 `docs/CONFIG_GUIDE.md` 配置指南文檔
  - 詳細說明主題、插件和命令的配置方法
  - 提供完整的代碼示例和實際案例
  - 說明 PATCH 方法的版本檢查和更新機制
  - 添加文檔引用到 README.md
- 更新插件配置結構
  - 添加插件 `directory` 鍵，與主題配置保持一致
  - 標準化插件路徑管理
  - 加強插件更新機制的版本檢查
  - 移除 `description` 欄位，添加 `request_schema` 欄位規範請求體格式
  - 支持按 HTTP 方法定義不同的請求體結構，提高安全性和可維護性
- 簡化命令配置結構
  - 簡化 `commands.json` 配置為 `"id": "file"` 格式
  - 採用通用參數格式 `*#ARG_N#*` 替代自定義參數
  - 提高命令腳本的靈活性和可重用性
- 改進首頁訪問邏輯
  - 修改根路徑(`/`)處理方式，直接顯示當前主題的 index.html
  - 移除原測試用的 JSON 響應信息
  - 添加主題目錄檢查與錯誤處理機制
  - 無需重定向即可直接訪問主題內容
- 修復系統處理器功能
  - 添加缺失的 gopsutil 依賴包
  - 實現系統資源監控功能
  - 添加負載平均值獲取方法
  - 修復 SystemHandler 結構和方法
  - 重新啟用系統信息和狀態 API
- 修復路由處理器配置問題
  - 修正 `AuthMiddleware` 函數接受 `*services.ConfigService` 類型參數
  - 更新 `routes.go` 使用正確的處理器方法名稱 (`LoginHandler` 而非 `Login`)
  - 保留功能完善的插件和命令 API 路由
  - 簡化路由配置，減少未定義方法的引用

### 2024 年 7 月 6 日

- 修正项目路由系统，移除冗余的路由处理机制
- 确保 routes.go 文件中的路由配置符合当前代码结构
- 修复 AuthMiddleware 相关参数不匹配问题

### 2024 年 7 月 1 日

- 修復路由和命令相關功能問題
  - 移除多餘的`api_routes.go`文件，消除函數重複聲明問題
  - 修復`CommandsConfig`類型錯誤，從`models.CommandsConfig`改為`*models.SystemCommandsConfig`
  - 添加缺失的`utils.SaveThemesConfig`函數，修復主題保存功能
  - 為`SystemCommandsConfig`結構體添加`GetCommandPath`方法，支持命令路徑查找
  - 重構`command_handler.go`中的 API 處理函數，使其兼容新的配置結構
  - 修復命令查詢邏輯，使用`Routes`成員訪問命令配置
  - 恢復使用原有的`SetupRoutes`函數處理路由配置
  - 維持 API 路由功能不變，確保原有 API 可正常使用
  - 清理冗餘代碼，提高系統穩定性

### 2024 年 6 月 30 日

- 重組腳本目錄結構並更改命名規則
  - 將`scripts/`目錄改為`plugins/`，用於存放插件相關腳本
  - 將`pkg/scripts/`目錄改為`commands/`，用於存放命令相關腳本
  - 保持原有腳本功能不變，僅調整目錄結構
  - 更新相關配置文件和代碼引用：
    - 更新`configs/commands.json`中的路徑引用
    - 將`ScriptsDir`常量更名為`CommandsDir`並設置值為`commands`
    - 更新`script_handler.go`文件名為`command_handler.go`並對應更新結構體名稱
    - 將`GetScriptPath`方法更名為`GetCommandPath`
    - 更新`route_loader.go`文件名為`command_loader.go`並對應更新相關結構體：
      - `RouteManager` → `CommandManager`
      - `RouteScript` → `CommandScript`
      - `LoadRoute` → `LoadCommand`
    - 更新配置相關代碼：
      - `RoutesConfig` → `SystemCommandsConfig`
      - `RouteConfig` → `CommandConfig`
      - `LoadRoutesConfig` → `LoadCommandsConfig`
    - 更新 API 路由相關代碼：
      - `routes.go` → `api_routes.go`
      - `SetupRoutes` → `SetupAPIRoutes`
    - 優化命令加載邏輯，使用固定的`/commands`路徑而非相對路徑
    - 刪除原始的`route_loader.go`文件，完全由新的`command_loader.go`文件替代
  - 標準化目錄結構：
    - `/plugins`: 插件相關腳本，如數據生成
    - `/commands`: 命令相關腳本，包含以下子目錄
      - `comment/`: 評論相關命令
      - `product/`: 產品管理命令
      - `user/`: 用戶管理命令
    - `/web`: 網頁相關文件和資源
  - 目錄路徑不再假設，而是使用固定的絕對路徑，提高代碼可讀性和可維護性
  - 代碼命名更加統一，明確區分插件和命令的相關操作

### 2024 年 6 月 29 日

- 修正 JSON 配置文件結構
  - 修正科技 lion 面板的 js 目錄為嵌套結構格式
  - 移除不存在的 images 目錄引用
  - 為 WhatsDifferent 插件添加 directory 和 structure 鍵
  - 優化插件資源文件結構映射
  - 標準化 JSON 配置文件格式
- 更新科技 lion 面板結構
  - 更新主題文件結構以符合實際目錄布局
  - 修正 js 目錄為嵌套結構格式
  - 移除不存在的 images 目錄引用
  - 正確映射 HTML 文件路徑
  - 修復樣式表引用路徑
  - 升級版本號至 1.0.1

### 2024 年 6 月 14 日

- 改進科技 lion 面板儀表板界面
  - 將系統狀態監控條移至側邊欄底部，隨側邊欄展開/收起
  - 增加側邊欄懸停自動展開功能
  - 添加側邊欄固定展開按鈕，支持長時間展開
  - 優化側邊欄收起時的圖標顯示
  - 改進系統狀態條的實時數據更新
  - 美化側邊欄展開/收起的過渡動畫效果

### 2024 年 6 月 13 日

- 優化科技 lion 面板界面按鈕功能
  - 重新設計黑暗模式切換開關，使用滑動式設計
  - 修復側邊欄展開/收起功能，添加側邊欄狀態記憶功能
  - 優化側邊欄項目結構，改善收起時的顯示效果
  - 添加深色模式樣式優化，提高暗色主題可讀性
  - 實現側邊欄切換時的平滑過渡效果

### 2024 年 6 月 12 日

- 修復科技 lion 面板儀表板問題
  - 替換缺失的 logo 圖片為 Font Awesome 圖標
  - 修復 sidebar 切換按鈕功能
  - 增強用戶下拉菜單功能
  - 修復暗黑模式切換功能
  - 優化 UI 樣式和布局，使用內聯樣式替代缺失樣式
  - 改進頁面元素的響應式布局

### 2024 年 6 月 8 日 - v1.5.2

- 規範化 API HTTP 方法使用
- 明確 PATCH 方法版本檢查和更新機制
- 對 PUT 和 PATCH 更新方式進行優化，只替換變化部分
- 檢查源連接 URL 與配置文件進行版本對比
- 添加示例主題、插件和命令在 example 目錄

### 2024 年 5 月 20 日 - v1.5.1

- 優化系統資源使用
- 修復插件管理功能的小錯誤
- 增強安全性：更新密碼加密方式
- 添加批量操作 API 功能

### 2024 年 4 月 12 日 - v1.5.0

- 新增主題系統
- 添加多語言支持
- 優化控制面板 UI/UX
- 增加數據庫備份和恢復功能

### 2024 年 3 月 5 日 - v1.4.2

- 修復插件系統安全漏洞
- 優化資源監控圖表
- 更新依賴項到最新版本

### 2024 年 2 月 15 日 - v1.4.1

- 增強日誌記錄功能
- 修復用戶權限問題
- 優化 HTTPS 配置流程

### 2024 年 1 月 20 日 - v1.4.0

- 添加插件系統
- 增強 API 功能
- 優化系統監控性能
- 新增命令行工具

### 2023 年 12 月 10 日 - v1.3.0

- 添加 Docker 容器管理
- 新增快捷操作面板
- 優化數據庫管理功能
- 改進文件管理器

### 2023 年 11 月 5 日 - v1.2.1

- 修復安全漏洞
- 改進性能監控
- 優化安裝腳本

### 2023 年 10 月 15 日 - v1.2.0

- 添加 SSH 密鑰管理
- 增強網站管理功能
- 優化系統設置界面

### 2023 年 9 月 20 日 - v1.1.0

- 新增網站管理功能
- 添加 SSL 證書自動化功能
- 優化用戶界面

### 2023 年 8 月 10 日 - v1.0.0

- 首次正式發布
- 基礎系統監控功能
- 服務管理功能
- 基本安全設置

## 貢獻

我們歡迎任何形式的貢獻，包括但不限於：

- 報告問題
- 提交功能請求
- 代碼貢獻
- 文檔改進

## 授權

本項目基於 MIT 許可證授權 - 詳情見 LICENSE 文件。
