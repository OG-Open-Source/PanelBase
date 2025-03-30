# PanelBase 主題與插件格式規範

本文檔詳細定義了 PanelBase 系統中主題安裝文件、插件安裝文件和命令文件的標準格式，供開發者參考使用。

## 目錄

- [主題安裝文件格式](#主題安裝文件格式)
  - [基本結構](#主題基本結構)
  - [結構說明](#主題結構說明)
  - [檔案結構](#主題檔案結構)
  - [示例](#主題示例)
- [插件安裝文件格式](#插件安裝文件格式)
  - [基本結構](#插件基本結構)
  - [結構說明](#插件結構說明)
  - [端點定義](#插件端點定義)
  - [示例](#插件示例)
- [命令文件格式](#命令文件格式)
  - [基本結構](#命令基本結構)
  - [元數據說明](#命令元數據說明)
  - [示例](#命令示例)
- [API 接口參考](#api接口參考)
  - [HTTP 方法使用規範](#http方法使用規範)
  - [主題相關 API](#主題相關api)
  - [插件相關 API](#插件相關api)
  - [命令相關 API](#命令相關api)

## 主題安裝文件格式

### 主題基本結構

主題安裝文件使用 JSON 格式，基本結構如下：

```json
{
  "theme": {
    "theme_id": {
      "name": "主題名稱",
      "authors": ["作者1", "作者2"],
      "version": "版本號",
      "description": "主題描述",
      "source_link": "主題源碼或更新鏈接",
      "directory": "主題目錄名",
      "structure": {
        // 檔案結構定義
      }
    }
  }
}
```

### 主題結構說明

| 欄位          | 類型          | 必填 | 說明                                             |
| ------------- | ------------- | ---- | ------------------------------------------------ |
| `theme`       | Object        | 是   | 主題容器對象                                     |
| `theme_id`    | String (Key)  | 是   | 主題的唯一標識符，同時作為對象的鍵名             |
| `name`        | String        | 是   | 主題的顯示名稱                                   |
| `authors`     | Array[String] | 是   | 主題開發者列表                                   |
| `version`     | String        | 是   | 版本號，遵循語義化版本規範 (Semantic Versioning) |
| `description` | String        | 是   | 主題的簡短描述                                   |
| `source_link` | String        | 否   | 主題的源碼或更新檢查的 URL                       |
| `directory`   | String        | 是   | 主題在系統中的安裝目錄名                         |
| `structure`   | Object        | 是   | 主題的檔案結構定義                               |

### 主題檔案結構

`structure` 欄位定義了主題包含的所有檔案及其下載 URL。結構可以包含嵌套的目錄：

```json
"structure": {
	"css": {
		"style.css": "https://example.com/theme/css/style.css"
	},
	"js": {
		"script.js": "https://example.com/theme/js/script.js"
	},
	"index.html": "https://example.com/theme/index.html"
}
```

系統會按照此結構將檔案下載到主題目錄下對應的路徑中。

### 主題示例

以下是一個完整的主題安裝文件示例：

```json
{
  "theme": {
    "kejilion": {
      "name": "科技lion面板",
      "authors": ["PanelBase Team", "科技lion"],
      "version": "0.1.0",
      "description": "轻量级Linux服务器运维管理面板",
      "source_link": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/kejilion.json",
      "directory": "kejilion",
      "structure": {
        "css": {
          "style.css": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/css/style.css"
        },
        "js": {
          "script.js": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/js/script.js",
          "nav.js": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/js/nav.js"
        },
        "docker.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/docker.html",
        "filemanager.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/filemanager.html",
        "index.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/index.html",
        "login.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/login.html",
        "market.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/market.html",
        "settings.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/settings.html",
        "website.html": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/web/kejilion/website.html"
      }
    }
  }
}
```

## 插件安裝文件格式

### 插件基本結構

插件安裝文件使用 JSON 格式，基本結構如下：

```json
{
  "plugin": {
    "plugin_id": {
      "name": "插件名稱",
      "authors": "插件作者",
      "version": "版本號",
      "description": "插件描述",
      "source_link": "插件源碼鏈接",
      "api_version": "API版本",
      "dependencies": {
        "依賴包名": "版本要求"
      },
      "endpoints": [
        {
          "path": "端點路徑",
          "methods": ["支援的HTTP方法"],
          "input_format": {
            // 輸入參數定義
          },
          "output_format": {
            // 輸出格式定義
          }
        }
      ]
    }
  }
}
```

### 插件結構說明

| 欄位           | 類型                   | 必填 | 說明                                             |
| -------------- | ---------------------- | ---- | ------------------------------------------------ |
| `plugin`       | Object                 | 是   | 插件容器對象                                     |
| `plugin_id`    | String (Key)           | 是   | 插件的唯一標識符，同時作為對象的鍵名             |
| `name`         | String                 | 是   | 插件的顯示名稱                                   |
| `authors`      | String / Array[String] | 是   | 插件開發者，可以是字符串或字符串數組             |
| `version`      | String                 | 是   | 版本號，遵循語義化版本規範 (Semantic Versioning) |
| `description`  | String                 | 是   | 插件的簡短描述                                   |
| `source_link`  | String                 | 是   | 插件的源碼位置                                   |
| `api_version`  | String                 | 是   | 插件使用的 API 版本（例如"v1"、"v2"）            |
| `dependencies` | Object                 | 否   | 插件依賴的外部庫和版本                           |
| `endpoints`    | Array[Object]          | 是   | 插件提供的 API 端點列表                          |

### 插件端點定義

每個端點對象包含以下欄位：

| 欄位            | 類型          | 必填 | 說明                                              |
| --------------- | ------------- | ---- | ------------------------------------------------- |
| `path`          | String        | 是   | 端點路徑（相對於插件基礎路徑）                    |
| `methods`       | Array[String] | 是   | 支援的 HTTP 方法（GET, POST, PUT, PATCH, DELETE） |
| `input_format`  | Object        | 否   | 輸入參數的格式定義                                |
| `output_format` | Object        | 否   | 輸出數據的格式定義                                |

`methods` 欄位指定了端點支援的 HTTP 方法，含義如下：

- `GET`: 檢索資源
- `POST`: 創建新資源
- `PUT`: 更新現有資源（完整替換）
- `PATCH`: 部分更新現有資源
- `DELETE`: 刪除資源

### 插件示例

以下是一個完整的插件安裝文件示例：

```json
{
  "plugin": {
    "whatsdifferent": {
      "name": "WhatsDifferent",
      "authors": "PanelBase Team",
      "version": "1.0.0",
      "description": "WhatsDifferent插件",
      "source_link": "https://github.com/OG-Open-Source/PanelBase/pkg/plugins/whatsdifferent",
      "api_version": "v1",
      "dependencies": {
        "golang.org/x/text": "0.23.0",
        "github.com/OG-Open-Source/WhatsDifferent": "1.2.4"
      },
      "endpoints": [
        {
          "path": "/diff",
          "methods": ["GET", "POST"],
          "input_format": {
            "original": "string",
            "new": "string"
          },
          "output_format": {
            "message": "string",
            "timestamp": "string"
          }
        },
        {
          "path": "/data",
          "methods": ["GET"],
          "input_format": {},
          "output_format": {
            "data": "array",
            "count": "number"
          }
        }
      ]
    }
  }
}
```

## 命令文件格式

### 命令基本結構

命令文件與主題和插件不同，不採用 JSON 文件作為安裝包，而是直接提供腳本文件。命令文件的元數據直接在腳本前 10 行內使用特殊標記定義。

腳本可以包含多個命令段落，每個段落都需要使用特定的元數據標記來定義其屬性。段落之間使用 `---` 分隔。

基本結構如下：

```bash
#!/bin/bash
# @command: shell
# @pkg_managers: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @authors: PanelBase Team
# @version: 1.0.0
# @description: Execute command
# @source_link: https://example.com/command.sh
# @execute_dir: /path/to/directory
# @allow_users: admin, editor

*#ARG_1#*
```

---

```go
// @command: golang
// @pkg_managers: apk, apt, opkg, pacman, yum, zypper, dnf
// @dependencies: null
// @authors: PanelBase Team
// @version: 1.0.0
// @description: Execute command
// @source_link: https://example.com/command.sh
// @execute_dir: /path/to/directory
// @allow_users: admin, editor

*#ARG_1#*
```

### 命令元數據說明

| 元數據標記      | 說明                                                                                                  |
| --------------- | ----------------------------------------------------------------------------------------------------- |
| `@command`      | 路由代號，用於識別該命令                                                                              |
| `@pkg_managers` | 支持的包管理器列表（Windows 使用 chocolatey, scoop, winget；macOS 使用 brew；使用 null 表示不限系統） |
| `@dependencies` | 所需的依賴包（會由 PanelBase 根據 @pkg_managers 自動安裝）                                            |
| `@authors`      | 命令作者                                                                                              |
| `@version`      | 版本號                                                                                                |
| `@description`  | 命令簡介                                                                                              |
| `@source_link`  | 命令來源鏈接                                                                                          |
| `@execute_dir`  | 命令執行目錄                                                                                          |
| `@allow_users`  | 允許執行該命令的用戶角色                                                                              |

參數在命令中使用 `*#ARG_N#*` 格式表示，其中 N 是參數的序號（從 1 開始）。例如，`*#ARG_1#*` 表示第一個參數。

### 命令示例

以下是一個完整的命令文件示例：

```bash
#!/bin/bash
# @command: install_nginx
# @pkg_managers: apt, yum, brew
# @dependencies: wget, curl
# @authors: PanelBase Team
# @version: 1.0.0
# @description: Install Nginx web server
# @source_link: https://example.com/install_nginx.sh
# @execute_dir: /tmp
# @allow_users: admin

# 檢查系統類型
if [ -f /etc/debian_release ]; then
	apt update
	apt install -y nginx
elif [ -f /etc/redhat-release ]; then
	yum install -y nginx
elif [ "$(uname)" == "Darwin" ]; then
	brew install nginx
else
	echo "Unsupported system"
	exit 1
fi

# 設置 Nginx 配置
cat > /etc/nginx/conf.d/*#ARG_1#*.conf << EOF
server {
	listen 80;
	server_name *#ARG_2#*;

	location / {
		root *#ARG_3#*;
		index index.html;
	}
}
EOF

# 重啟 Nginx
if [ -f /etc/debian_release ]; then
	systemctl restart nginx
elif [ -f /etc/redhat-release ]; then
	systemctl restart nginx
elif [ "$(uname)" == "Darwin" ]; then
	brew services restart nginx
fi

echo "Nginx configured for *#ARG_2#* with root directory *#ARG_3#*"
```

## API 接口參考

### HTTP 方法使用規範

PanelBase 系統對 HTTP 方法的使用進行了統一規範，確保 API 接口的一致性：

- **GET**:

  - 無請求體：表示查看列表資源
  - 有請求體 `{"id":"target_id"}`：表示查看位於本機指定的目標資源

- **POST**:

  - 必定帶有請求體
  - 安裝資源：`{"url":"source_link"}`
  - 插件特殊操作：`{"id":"{plugin_id}","status":"start/stop"}`
  - 主題切換：`{"id":"{theme_id}","status":"switch"}`

- **PUT**:

  - 不會有請求體
  - 表示更新內容列表中的全部內容
  - 系統將檢查所有已安裝項目的 source_link，並對每個項目進行版本比對，如有更新則下載新版本

- **PATCH**:

  - 必定帶有請求體，只需包含 `{"id":"target_id"}`
  - 只會更新位於本機且指定的目標
  - 系統將根據目標的 source_link 檢查版本差異，如有更新則下載新版本
  - 這種方法不需要提供任何其他參數，系統會自動查找目標的 source_link 進行版本檢查

- **DELETE**:
  - 無請求體：表示刪除列表中的全部內容
  - 有請求體 `{"id":"target_id"}`：將會刪除位於本機且指定的目標

請注意，目標指的是一個物件的本身，如插件或主題等，系統將對整個目標進行處理，不會出現目標只有部分受到處理的情況。

#### 版本檢查與更新機制

當執行 PUT 或 PATCH 方法時，系統會執行下列版本檢查流程：

- **主題**：系統通過比對本機主題的 version 和 source_link 指向的遠程主題檔案中的 version 判斷是否需要更新
- **插件**：與主題相同，通過 version 和 source_link 比對判斷更新
- **命令**：系統解析腳本中的 `# @version` 和 `# @source_link` 標記，與遠程版本比對確定是否更新

版本比對成功後，更新流程如下：

1. 系統下載新版本的元數據和配置結構
2. 根據結構定義，下載需要更新的文件
3. 僅替換或新增變更的部分，保留舊版產生的其他目錄文件
4. 更新本地配置中的版本信息

這種方式確保僅更新必要的內容，降低帶寬使用，同時保留用戶可能的本地修改和生成的數據文件。

### 主題相關 API

主題相關 API 均需透過 `/api/v1/theme` 進行訪問：

- **GET /api/v1/theme**

  - 無請求體：獲取所有主題列表
  - 請求體 `{"id":"theme_id"}`：獲取指定主題詳情

- **POST /api/v1/theme**

  - 請求體 `{"url":"https://example.com/theme.json"}`：安裝新主題
  - 請求體 `{"id":"theme_id","status":"switch"}`：切換當前使用的主題

- **PUT /api/v1/theme**

  - 無請求體：更新所有已安裝主題

- **PATCH /api/v1/theme**

  - 請求體 `{"id":"theme_id"}`：更新指定主題

- **DELETE /api/v1/theme**
  - 無請求體：刪除所有非當前使用的主題
  - 請求體 `{"id":"theme_id"}`：刪除指定主題

示例：

```bash
# 獲取所有主題
curl -X GET -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/theme

# 獲取指定主題
curl -X GET -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"kejilion"}' http://localhost:8080/api/v1/theme

# 安裝新主題
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"url":"https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/example/ex_kejilion.json"}' http://localhost:8080/api/v1/theme

# 切換當前使用的主題
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"kejilion","status":"switch"}' http://localhost:8080/api/v1/theme

# 更新全部主題
curl -X PUT -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/theme

# 更新指定主題
curl -X PATCH -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"kejilion"}' http://localhost:8080/api/v1/theme

# 刪除所有非當前主題
curl -X DELETE -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/theme

# 刪除指定主題
curl -X DELETE -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"kejilion"}' http://localhost:8080/api/v1/theme
```

### 插件相關 API

插件相關 API 均需透過 `/api/v1/plugins` 進行訪問：

- **GET /api/v1/plugins**

  - 無請求體：獲取所有插件列表
  - 請求體 `{"id":"plugin_id"}`：獲取指定插件詳情

- **POST /api/v1/plugins**

  - 請求體 `{"url":"https://example.com/plugin.json"}`：安裝新插件
  - 請求體 `{"id":"plugin_id","status":"start"}`：啟動插件
  - 請求體 `{"id":"plugin_id","status":"stop"}`：停止插件

- **PUT /api/v1/plugins**

  - 無請求體：更新所有已安裝插件

- **PATCH /api/v1/plugins**

  - 請求體 `{"id":"plugin_id"}`：更新指定插件

- **DELETE /api/v1/plugins**
  - 無請求體：刪除所有插件
  - 請求體 `{"id":"plugin_id"}`：刪除指定插件

插件 API 調用則使用：

- **Any /api/v1/plugins/{plugin_id}/{api_path}**：調用指定插件的指定 API 路徑

示例：

```bash
# 獲取所有插件
curl -X GET -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/plugins

# 獲取指定插件
curl -X GET -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"whatsdifferent"}' http://localhost:8080/api/v1/plugins

# 安裝新插件
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"url":"https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/example/ex_whatsdifferent.json"}' http://localhost:8080/api/v1/plugins

# 啟動插件
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"whatsdifferent","status":"start"}' http://localhost:8080/api/v1/plugins

# 停止插件
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"whatsdifferent","status":"stop"}' http://localhost:8080/api/v1/plugins

# 更新全部插件
curl -X PUT -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/plugins

# 更新指定插件
curl -X PATCH -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"whatsdifferent"}' http://localhost:8080/api/v1/plugins

# 刪除所有插件
curl -X DELETE -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/plugins

# 刪除指定插件
curl -X DELETE -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"whatsdifferent"}' http://localhost:8080/api/v1/plugins

# 調用插件API (GET方法)
curl -X GET -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/plugins/whatsdifferent/diff?original=text1&new=text2

# 調用插件API (POST方法)
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"original":"text1","new":"text2"}' http://localhost:8080/api/v1/plugins/whatsdifferent/diff
```

### 命令相關 API

命令相關 API 均需透過 `/api/v1/commands` 進行訪問：

- **GET /api/v1/commands**

  - 無請求體：獲取所有命令列表
  - 請求體 `{"id":"command_id"}`：獲取指定命令詳情

- **POST /api/v1/commands**

  - 請求體 `{"url":"https://example.com/command.sh"}`：安裝新命令

- **PUT /api/v1/commands**

  - 無請求體：更新所有已安裝命令

- **PATCH /api/v1/commands**

  - 請求體 `{"id":"command_id"}`：更新指定命令

- **DELETE /api/v1/commands**
  - 無請求體：刪除所有命令
  - 請求體 `{"id":"command_id"}`：刪除指定命令

命令執行則使用：

- **POST /api/v1/execute**：執行命令（批量命令執行）

示例：

```bash
# 獲取所有命令
curl -X GET -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/commands

# 獲取指定命令
curl -X GET -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"install_nginx"}' http://localhost:8080/api/v1/commands

# 安裝新命令
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"url":"https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/example/ex_install_nginx.sh"}' http://localhost:8080/api/v1/commands

# 更新全部命令
curl -X PUT -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/commands

# 更新指定命令
curl -X PATCH -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"install_nginx"}' http://localhost:8080/api/v1/commands

# 刪除所有命令
curl -X DELETE -H "Authorization: Bearer {your_token}" http://localhost:8080/api/v1/commands

# 刪除指定命令
curl -X DELETE -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '{"id":"install_nginx"}' http://localhost:8080/api/v1/commands

# 執行命令（單個命令）
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '[{"command":"install_nginx","args":["mysite","example.com","/var/www/html"]}]' http://localhost:8080/api/v1/execute

# 執行命令（批量命令）
curl -X POST -H "Authorization: Bearer {your_token}" -H "Content-Type: application/json" -d '[
	{"command":"install_nginx","args":["mysite","example.com","/var/www/html"]},
	{"command":"update_firewall","args":["allow","80"]}
]' http://localhost:8080/api/v1/execute
```
