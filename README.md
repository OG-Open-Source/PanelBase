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
- 密碼：查看 `/root/panelbase_password.txt`

## 配置指南

詳細的配置說明請參閱以下文檔：

- [配置指南](docs/CONFIG_GUIDE.md)：詳細說明主題、插件和命令的配置方法
- [詳細信息](docs/INFO.md)：主題與插件格式規範

## API 使用

PanelBase 提供了完整的 API 接口，支持以下操作：

### 主題相關 API

- **GET /api/v1/theme**: 獲取所有主題或特定主題信息

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" http://your-server-ip:8080/api/v1/theme
  curl -H "Authorization: Bearer YOUR_TOKEN" -X GET -d '{"id":"theme_id"}' http://your-server-ip:8080/api/v1/theme
  ```

- **POST /api/v1/theme**: 安裝新主題或切換當前主題

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"url":"theme_source_url"}' http://your-server-ip:8080/api/v1/theme
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"id":"theme_id","status":"switch"}' http://your-server-ip:8080/api/v1/theme
  ```

- **PUT /api/v1/theme**: 更新所有已安裝主題

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PUT http://your-server-ip:8080/api/v1/theme
  ```

- **PATCH /api/v1/theme**: 更新特定主題

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PATCH -d '{"id":"theme_id"}' http://your-server-ip:8080/api/v1/theme
  ```

- **DELETE /api/v1/theme**: 删除所有非當前主題或特定主題
  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE http://your-server-ip:8080/api/v1/theme
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE -d '{"id":"theme_id"}' http://your-server-ip:8080/api/v1/theme
  ```

### 插件相關 API

- **GET /api/v1/plugin**: 獲取所有插件或特定插件信息

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" http://your-server-ip:8080/api/v1/plugin
  curl -H "Authorization: Bearer YOUR_TOKEN" -X GET -d '{"id":"plugin_id"}' http://your-server-ip:8080/api/v1/plugin
  ```

- **POST /api/v1/plugin**: 安裝新插件

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X POST -d '{"url":"plugin_source_url"}' http://your-server-ip:8080/api/v1/plugin
  ```

- **PUT /api/v1/plugin**: 更新所有已安裝插件

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PUT http://your-server-ip:8080/api/v1/plugin
  ```

- **PATCH /api/v1/plugin**: 更新特定插件

  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X PATCH -d '{"id":"plugin_id"}' http://your-server-ip:8080/api/v1/plugin
  ```

- **DELETE /api/v1/plugin**: 删除所有插件或特定插件
  ```bash
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE http://your-server-ip:8080/api/v1/plugin
  curl -H "Authorization: Bearer YOUR_TOKEN" -X DELETE -d '{"id":"plugin_id"}' http://your-server-ip:8080/api/v1/plugin
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
