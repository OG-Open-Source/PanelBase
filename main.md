# PanelBase

## 1. 專案概述

`PanelBase` 專案旨在建立一個靈活且可擴充的後端服務，用於管理多步驟腳本執行和主題/路由管理。此專案使用 Go 語言開發，並遵循 Go 語言的最佳實踐。

## 2. 文件結構詳解

```text
PanelBase
├── cmd
│   └── panelbase
│       └── main.go
├── internal
│   ├── commands
│   │   ├── run.sh
│   │   ├── time.go
│   │   └── date.py
│   ├── config
│   │   ├── config.go
│   │   ├── routes.json
│   │   └── themes.json
│   ├── handlers
│   │   └── external.go
│   └── themes
│       ├── kejilion
│       │   └── index.html
│       └── panelbase
│           └── index.html
├── pkg
│   └── utils
│       ├── logger.go
│       ├── route.go
│       └── theme.go
├── web
│   └── static
│       └── favicon.png
├── .env
├── build.sh
├── deploy.sh
└── go.mod
```

### 2.1. cmd 目錄

- `cmd/panelbase/main.go`：
  - 專案的入口點，負責初始化配置、設定路由和啟動 HTTP 伺服器。
  - 使用 `handlers.SetupRoutes()` 啟動路由。

### 2.2. internal 目錄

- `internal`：存放私有程式碼，防止外部包引用。

  - `internal/scripts`：
    - **存放可執行檔，可以是 Shell 腳本、Go 程式或 Python 腳本等。**
    - **這些可執行檔將根據 API 請求的配置，由 `pkg/utils/route.go` 定位並在 `/tmp` 目錄下執行。**
    - `run.sh`：根據變量的指令執行的腳本。
    - `time.go`：提供時間的腳本。
    - `date.py`：提供日期的腳本。
  - `internal/config`：
    - `config.go`：負責讀取和解析應用程式的配置。
    - `routes.json`：存放路由配置，由 `pkg/utils/route.go` 管理。
      - 格式：`{"路由名稱": "腳本名稱"}`
      - 例如：`{"time": "time.go", "run": "run.sh", "date": "date.py"}`
    - `themes.json`：存放主題配置，由 `pkg/utils/theme.go` 管理。
      - 格式：`{"主題名稱": {"name": "主題顯示名稱", "authors": "作者", "version": "版本", "description": "描述"}}`
      - 例如：`{"panelbase": {"name": "PanelBase Base", "authors": "PanelBase Team", "version": "1.0.0", "description": "Basic style theme for PanelBase"}, "kejilion": {"name": "科技lion官方主题", "authors": "Kejilion", "version": "1.0.0", "description": "科技lion官方的PanelBase主题"}}`
  - `internal/handlers`：
    - `external.go`：作為唯一的對外 API 接口，負責將請求導向 `pkg/utils/route.go` 和 `pkg/utils/theme.go`。
  - `internal/themes`：
    - 存放主題相關的 HTML 檔案。

### 2.3. pkg 目錄

- `pkg`：存放可重用的公共程式碼。

  - `pkg/utils`：
    - `logger.go`：提供日誌記錄功能。
      - 日誌格式：`[日誌級別] 2023-10-27T10:00:00.123Z | 訊息 main.go:123`
      - 日誌級別字數相等，例如：`DEBUG`、`INFO `、`WARN `、`ERROR`、`FATAL`。
      - 使用 UTC 時間戳記。
      - 提供 `Debug`、`Info`、`Warn`、`Error`、`Fatal` 等日誌級別的函式。
      - 使用 `runtime.Caller` 獲取檔案名稱和行號。
    - `route.go`：負責處理 API 請求，執行多步驟腳本和路由管理。
      - 執行多個腳本：接收包含 `commands` 陣列的 API 請求，每個元素都是一個指令物件，包含 `script`（腳本名稱）和 `args`（變數值列表）。
      - 依序執行 `commands` 中的每個指令。
      - 在執行每個指令之前，使用正則表達式將 `*#ARG_n#*` 替換為變數值，並將前一個指令的結果作為參數傳遞給腳本。
      - 將所有指令的輸出合併，作為 API 的最終回應。
      - 路由管理：
        - `install`, `update`, `metadata`: 接收 JSON `{"url": "主題目錄的連結"}`。
        - `delete`: 接收 JSON `{"name": "路由名稱"}`，路由名稱從 `routes.json` 的鍵中選取。
    - `theme.go`：負責管理 `themes.json`，提供主題相關的 API 端點。
      - 負責讀取 `themes.json` 檔案，解析其中的主題配置，並將其儲存在記憶體中。
      - 提供 API 端點，允許外部請求新增、更新、刪除和查詢主題配置。
      - 在應用程式啟動時，載入 `themes.json`，並在主題變更時，更新記憶體中的主題配置。
      - 提供函式，允許 `external.go` 根據請求的主題名稱，查找相應的主題檔案。
      - 主題管理：
        - `install`, `update`, `metadata`: 接收 JSON `{"url": "主題目錄的連結"}`。
        - `delete`: 接收 JSON `{"name": "主題名稱"}`，主題名稱從 `themes.json` 的鍵中選取。

### 2.4. web 目錄

- `web/static`：
  - 存放靜態資源，例如 `favicon.png`。

### 2.5. 其他檔案

- `.env`：存放環境變數。且至少存在 IP PORT ENTRY 三個變數。
- `go.mod`：管理 Go 語言的依賴。

## 3. API 設計

- `external.go` 作為唯一的對外 API 接口。
- 多步驟腳本執行 API：
  - 請求方法：**POST**
  - 請求路徑：`/ENTRY/execute`
  - 請求體：

```json
{
  "commands": [
    {
      "script": "script1.sh",
      "args": ["value1_for_script1", "value2_for_script1"]
    },
    {
      "script": "script2.py",
      "args": ["value1_for_script2"]
    },
    {
      "script": "script3.go",
      "args": ["value1_for_script3", "value2_for_script3", "value3_for_script3"]
    }
  ]
}
```

- 核心 API：
  - /ENTRY/execute: POST, 執行腳本。
  - /ENTRY/theme/install: POST, 安裝主題，接收 {"url": "主題目錄的連結"}。
  - /ENTRY/theme/update: POST, 更新主題，接收 {"url": "主題目錄的連結"}。
  - /ENTRY/theme/metadata: GET, 查詢主題配置。
  - /ENTRY/theme/delete: POST, 刪除主題，接收 {"name": "主題名稱"}。
  - /ENTRY/route/install: POST, 安裝路由，接收 {"url": "腳本的連結"}。
  - /ENTRY/route/update: POST, 更新路由，接收 {"url": "腳本的連結"}。
  - /ENTRY/route/metadata: GET, 查詢路由配置。
  - /ENTRY/route/delete: POST, 刪除路由，接收 {"name": "路由名稱"}。
  - /ENTRY/ws-execute: WebSocket, 實時顯示腳本輸出。

## 4. 開發指南

- 設定開發環境。
- 使用 `build.sh` 構建應用程式。
- 使用 `go run cmd/panelbase/main.go` 執行應用程式。
- 使用 `external.go` 定義 API 接口。
- 使用 `pkg/utils/route.go` 處理腳本執行和路由管理。
- 使用 `pkg/utils/theme.go` 管理主題。

## 5. 部署指南

- 配置部署環境。
- 使用 `deploy.sh` 部署應用程式。
- 監控和管理應用程式。

## 6. 文件格式

路由文件範例

```text
# @script: 路由名稱
# @pkg_managers: 支持的包管理器，遇到不支持的系統時，禁止執行
# @dependencies: 需要的套件，會要求先安裝後，方能執行，否則禁止執行
# @authors: 作者名
# @version: 版本
# @description: 介紹
```

其中的 # 可以替換成 //，以適用於不同的程式語言(sh py)

主題文件範例

```json
{
  "theme": "主題被選取的名稱，同時為 /internal/themes 目錄下的目錄名",
  "name": "主題被顯示的名稱",
  "authors": "作者名",
  "version": "主題的版本",
  "description": "主題的介紹",
  "index.html": "http://example.com/download/index.html",
  "js/scripts.js": "http://example.com/download/scripts.js",
  "欲下載的文件名或目錄下文件名": "連結"
}
```

## 7. 開發指南

變量是屬於自定義的變量，使用 *#ARG_1#* 這與 $1 的意思相同，並且無法自訂變量，除非在腳本內自行設定 value1="*#ARG_1#*" 這樣就可以讓 $1 變成 value1

補充其 ASCII 流程圖，如下：
http://IP:PORT/ENTRY/{route/theme}/{install, update, delete} (POST) --> /internal/handlers/external.go --> /pkg/utils/{route.go/theme.go} --> 執行對應操作(install, update 獲取 JSON 儲存體 '{"url":"http://example.com/fulfill.file"}'，delete 獲取 JSON 儲存體 '{"name":"根據 routes.json 中所對應的路由名稱"}') --> 輸出結果

http://IP:PORT/ENTRY/{route/theme}/metadata (GET) --> /internal/handlers/external.go --> /pkg/utils/{route.go/theme.go} --> 獲取連結(JSON 儲存體 '{"url":"http://example.com/fulfill.file"}'，再根據其路由文件元數據顯示 @script @pkg_managers @dependencies @authors @version @description 這 6 項元數據，主題則是一個文件且包含 theme name authors version description 這 5 項元數據，並且格式如上) --> 輸出結果
