# PanelBase 配置指南

本文檔提供了配置 PanelBase 系統中主題、插件和命令的詳細說明，包括文件結構、配置選項和最佳實踐。

## 目錄

- [文件路徑規範](#文件路徑規範)
- [API 回應格式規範](#api-回應格式規範)
- [主題配置](#主題配置)
  - [themes.json 文件](#themesjson-文件)
  - [主題文件結構](#主題文件結構)
  - [自定義主題開發](#自定義主題開發)
  - [主題配置實例](#主題配置實例)
- [插件配置](#插件配置)
  - [plugins.json 文件](#pluginsjson-文件)
  - [插件開發規範](#插件開發規範)
  - [插件 API 接口](#插件-api-接口)
  - [插件開發實例](#插件開發實例)
- [命令配置](#命令配置)
  - [commands.json 文件](#commandsjson-文件)
  - [命令腳本開發](#命令腳本開發)
  - [參數定義與驗證](#參數定義與驗證)
  - [命令開發實例](#命令開發實例)
- [版本檢查與更新機制](#版本檢查與更新機制)
  - [PATCH 方法的版本驗證](#patch-方法的版本驗證)
  - [更新策略與優化](#更新策略與優化)
  - [更新流程示例](#更新流程示例)

## 文件路徑規範

PanelBase 使用以下目錄結構來組織各種配置和資源文件：

```
.
├── configs/                  # 配置文件目錄
│   ├── config.yaml           # 主配置文件
│   ├── users.json            # 用戶配置
│   ├── themes.json           # 主題配置
│   ├── plugins.json          # 插件配置
│   └── commands.json         # 命令配置
├── web/                      # 網頁資源目錄
│   ├── assets/               # 共享靜態資源
│   ├── default/              # 默認主題
│   └── [theme_directory]/    # 其他主題目錄
├── plugins/                  # 插件目錄
│   └── [plugin_directory]/   # 各插件目錄
└── commands/                 # 命令目錄
		├── user/                 # 用戶相關命令
		├── system/               # 系統相關命令
		└── [category]/           # 其他分類命令
```

## API 回應格式規範

PanelBase 系統中所有 API 回應均採用統一的 JSON 格式，確保前端處理的一致性和可預測性。

### 基本格式

```json
{
  "status": "success", // 或 "failure"
  "message": "操作成功", // 或其他根據內容自訂的消息
  "data": {
    // 可選，由內容自訂的數據對象
  }
}
```

### 欄位說明

| 欄位      | 類型   | 說明                                            |
| --------- | ------ | ----------------------------------------------- |
| `status`  | string | 必填，操作結果狀態，值為 "success" 或 "failure" |
| `message` | string | 必填，操作結果的描述信息                        |
| `data`    | object | 可選，返回的數據內容，根據 API 不同而不同       |

### 成功回應示例

```json
{
  "status": "success",
  "message": "用戶創建成功",
  "data": {
    "user_id": "12345",
    "username": "new_user",
    "created_at": "2024-07-08T12:34:56Z"
  }
}
```

### 失敗回應示例

```json
{
  "status": "failure",
  "message": "用戶名已存在",
  "data": {
    "error_code": "USER_EXISTS",
    "provided_username": "existing_user"
  }
}
```

### 注意事項

1. **一致性**: 所有 API 接口必須遵循此格式，包括插件、主題和命令 API
2. **語言適配**: `message`欄位應根據系統設置的語言進行本地化
3. **資料層級**: `data`對象中應避免深層嵌套，保持扁平化結構
4. **錯誤信息**: 失敗回應中應提供足夠的信息以便排查問題
5. **安全考慮**: 避免在錯誤消息中暴露敏感信息

所有開發者在擴展 PanelBase 功能時必須遵循此規範，確保 API 回應的一致性和可用性。

## 主題配置

### themes.json 文件

`themes.json` 文件位於 `/configs` 目錄下，用於定義所有已安裝的主題及當前使用的主題。

基本結構：

```json
{
  "current_theme": "default_theme",
  "themes": {
    "default_theme": {
      "name": "Default Theme",
      "authors": "PanelBase Team",
      "version": "1.0.0",
      "description": "Default theme for PanelBase",
      "source_link": "https://github.com/OG-Open-Source/PanelBase",
      "directory": "default"
    },
    "kejilion": {
      "name": "科技lion面板",
      "authors": ["PanelBase Team", "科技lion"],
      "version": "1.0.1",
      "description": "轻量级Linux服务器运维管理面板",
      "source_link": "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/kejilion.json",
      "directory": "kejilion"
    }
  }
}
```

配置項說明：

| 欄位            | 說明                                   |
| --------------- | -------------------------------------- |
| `current_theme` | 當前使用的主題 ID                      |
| `themes`        | 包含所有已安裝主題的對象               |
| `name`          | 主題名稱                               |
| `authors`       | 主題作者（字符串或數組）               |
| `version`       | 主題版本                               |
| `description`   | 主題描述                               |
| `source_link`   | 主題源碼或更新鏈接                     |
| `directory`     | 主題文件存放的目錄名（相對於 `/web/`） |

### 主題文件結構

每個主題都應包含以下基本文件：

1. **index.html** - 主頁模板
2. **login.html** - 登錄頁面模板

這些文件將存放在 `/web/[theme_directory]/` 目錄下。

### 自定義主題開發

開發自定義主題時，建議遵循以下最佳實踐：

1. **響應式設計**：確保主題在不同設備上都能正常顯示
2. **模塊化結構**：將 CSS 和 JS 分類存放在對應的子目錄中
3. **兼容性**：確保與 PanelBase API 接口兼容
4. **版本控制**：遵循語義化版本規範

### 主題配置實例

以下提供一個完整的主題配置實例，展示如何從安裝到使用的全過程：

#### 1. 創建主題 JSON 安裝文件

```json
{
  "theme": {
    "custom_theme": {
      "name": "自定義主題",
      "authors": "Your Name",
      "version": "1.0.0",
      "description": "自定義面板主題，現代設計風格",
      "source_link": "https://example.com/custom_theme.json",
      "directory": "custom_theme",
      "structure": {
        "css": {
          "style.css": "https://example.com/themes/custom_theme/css/style.css",
          "dark-mode.css": "https://example.com/themes/custom_theme/css/dark-mode.css"
        },
        "js": {
          "script.js": "https://example.com/themes/custom_theme/js/script.js",
          "charts.js": "https://example.com/themes/custom_theme/js/charts.js"
        },
        "img": {
          "logo.svg": "https://example.com/themes/custom_theme/img/logo.svg",
          "favicon.ico": "https://example.com/themes/custom_theme/img/favicon.ico"
        },
        "index.html": "https://example.com/themes/custom_theme/index.html",
        "login.html": "https://example.com/themes/custom_theme/login.html",
        "dashboard.html": "https://example.com/themes/custom_theme/dashboard.html",
        "settings.html": "https://example.com/themes/custom_theme/settings.html"
      }
    }
  }
}
```

#### 2. `themes.json` 更新

安裝主題後，`themes.json` 將更新為：

```json
{
  "current_theme": "custom_theme",
  "themes": {
    "default_theme": {
      "name": "Default Theme",
      "authors": "PanelBase Team",
      "version": "1.0.0",
      "description": "Default theme for PanelBase",
      "source_link": "https://github.com/OG-Open-Source/PanelBase",
      "directory": "default"
    },
    "custom_theme": {
      "name": "自定義主題",
      "authors": "Your Name",
      "version": "1.0.0",
      "description": "自定義面板主題，現代設計風格",
      "source_link": "https://example.com/custom_theme.json",
      "directory": "custom_theme"
    }
  }
}
```

## 插件配置

### plugins.json 文件

`plugins.json` 文件位於 `/configs` 目錄下，用於定義所有已安裝的插件。

基本結構：

```json
{
  "plugins": {
    "hello_plugin": {
      "name": "Hello Plugin",
      "author": "PanelBase Team",
      "version": "1.0.0",
      "description": "A simple hello world plugin",
      "source_link": "https://github.com/OG-Open-Source/PanelBase/plugins/hello_plugin.json",
      "directory": "hello_plugin",
      "api_version": "v1",
      "endpoints": [
        {
          "path": "/hello",
          "methods": ["GET"],
          "request_schema": {}
        },
        {
          "path": "/echo",
          "methods": ["POST", "PUT"],
          "request_schema": {
            "message": "string",
            "timestamp": "number",
            "options": "object"
          }
        }
      ]
    },
    "user_manager": {
      "name": "用户管理插件",
      "author": "PanelBase Team",
      "version": "1.2.0",
      "description": "用户管理和权限控制插件",
      "source_link": "https://github.com/OG-Open-Source/PanelBase/plugins/user_manager.json",
      "directory": "user_manager",
      "api_version": "v1",
      "endpoints": [
        {
          "path": "/users",
          "methods": ["GET", "POST", "PUT", "PATCH", "DELETE"],
          "request_schema": {
            "GET": {},
            "POST": {
              "username": "string",
              "email": "string",
              "password": "string",
              "role": "string"
            },
            "PUT": {
              "id": "string",
              "username": "string",
              "email": "string",
              "role": "string"
            },
            "PATCH": {
              "id": "string"
            },
            "DELETE": {
              "id": "string"
            }
          }
        }
      ]
    }
  }
}
```

配置項說明：

| 欄位             | 說明                                       |
| ---------------- | ------------------------------------------ |
| `plugins`        | 包含所有已安裝插件的對象                   |
| `name`           | 插件名稱                                   |
| `author`         | 插件作者                                   |
| `version`        | 插件版本                                   |
| `description`    | 插件描述                                   |
| `source_link`    | 插件源碼或更新鏈接                         |
| `directory`      | 插件文件存放的目錄名（相對於 `/plugins/`） |
| `api_version`    | 插件使用的 API 版本                        |
| `endpoints`      | 插件提供的 API 端點列表                    |
| `path`           | 端點路徑（相對於插件基礎路徑）             |
| `methods`        | 支援的 HTTP 方法                           |
| `request_schema` | 請求體格式規範，可以按 HTTP 方法區分       |

**注意**：

1. HTTP 方法只允許使用 GET、POST、PUT、PATCH 和 DELETE。
2. `request_schema` 定義了請求體的格式規範，確保插件只接收所需的資料內容：
   - 對於簡單端點，可以使用單一結構定義所有方法的請求格式
   - 對於複雜端點，可以按 HTTP 方法區分不同的請求結構
   - 支持的數據類型：`string`、`number`、`boolean`、`object`、`array`
   - 空對象 `{}` 表示無需請求參數或僅接受查詢參數
   - `GET` 請求通常使用查詢參數而非請求體，因此其 `request_schema` 通常為空對象
   - `PATCH` 請求通常只需提供資源 ID 和變更的欄位

### 插件開發規範

插件開發需遵循以下規範：

1. **API 優先**：所有功能都應通過 API 接口提供
2. **版本控制**：使用語義化版本號，並在 `source_link` 中提供更新檢查機制
3. **錯誤處理**：提供清晰的錯誤信息和狀態碼
4. **安全性**：防止注入攻擊和越權訪問

### 插件 API 接口

插件 API 路徑格式為：`/api/v1/plugins/{plugin_id}/{endpoint_path}`

例如，對於 `hello_plugin` 插件的 `/hello` 端點，完整的 API 路徑是：

```
/api/v1/plugins/hello_plugin/hello
```

### 插件開發實例

下面展示一個簡單的統計插件開發實例：

#### 1. 定義插件 JSON 配置

```json
{
  "plugin": {
    "stats_plugin": {
      "name": "System Statistics Plugin",
      "author": "Your Name",
      "version": "1.0.0",
      "description": "Advanced system statistics and monitoring",
      "source_link": "https://example.com/stats_plugin.json",
      "directory": "stats_plugin",
      "api_version": "v1",
      "endpoints": [
        {
          "path": "/summary",
          "methods": ["GET"],
          "request_schema": {}
        },
        {
          "path": "/cpu",
          "methods": ["GET"],
          "request_schema": {}
        },
        {
          "path": "/memory",
          "methods": ["GET"],
          "request_schema": {}
        },
        {
          "path": "/disk",
          "methods": ["GET"],
          "request_schema": {}
        },
        {
          "path": "/network",
          "methods": ["GET"],
          "request_schema": {}
        },
        {
          "path": "/history",
          "methods": ["GET", "POST", "DELETE"],
          "request_schema": {
            "GET": {
              "from": "string",
              "to": "string",
              "type": "string"
            },
            "POST": {
              "type": "string",
              "data": "object",
              "timestamp": "number"
            },
            "DELETE": {
              "from": "string",
              "to": "string",
              "type": "string"
            }
          }
        }
      ]
    }
  }
}
```

#### 2. 插件主處理文件

在 `/plugins/stats_plugin/main.go` 中實現插件功能：

```go
package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// 插件入口點，初始化插件
func Initialize() map[string]interface{} {
	return map[string]interface{}{
		"name":    "System Statistics Plugin",
		"version": "1.0.0",
		"routes": map[string]interface{}{
			"/summary": HandleSummary,
			"/cpu":     HandleCPU,
			"/memory":  HandleMemory,
			"/disk":    HandleDisk,
			"/network": HandleNetwork,
			"/history": HandleHistory,
		},
	}
}

// HandleSummary 處理系統概覽請求
func HandleSummary(req map[string]interface{}) map[string]interface{} {
	// 收集各項系統信息
	hostInfo, _ := host.Info()
	cpuInfo, _ := cpu.Info()
	cpuPercent, _ := cpu.Percent(time.Second, false)
	memInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")

	// 返回概覽數據
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"host": map[string]interface{}{
				"hostname":       hostInfo.Hostname,
				"os":             hostInfo.OS,
				"platform":       hostInfo.Platform,
				"platform_version": hostInfo.PlatformVersion,
				"uptime":         hostInfo.Uptime,
			},
			"cpu": map[string]interface{}{
				"model":   cpuInfo[0].ModelName,
				"cores":   runtime.NumCPU(),
				"usage":   cpuPercent[0],
			},
			"memory": map[string]interface{}{
				"total":     memInfo.Total,
				"used":      memInfo.Used,
				"available": memInfo.Available,
				"percent":   memInfo.UsedPercent,
			},
			"disk": map[string]interface{}{
				"total":     diskInfo.Total,
				"used":      diskInfo.Used,
				"available": diskInfo.Free,
				"percent":   diskInfo.UsedPercent,
			},
		},
	}
}

// HandleCPU 處理CPU詳細信息請求
func HandleCPU(req map[string]interface{}) map[string]interface{} {
	// 實現CPU詳細信息收集邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			// CPU詳細信息
		},
	}
}

// HandleMemory 處理內存詳細信息請求
func HandleMemory(req map[string]interface{}) map[string]interface{} {
	// 實現內存詳細信息收集邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			// 內存詳細信息
		},
	}
}

// HandleDisk 處理磁盤詳細信息請求
func HandleDisk(req map[string]interface{}) map[string]interface{} {
	// 實現磁盤詳細信息收集邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			// 磁盤詳細信息
		},
	}
}

// HandleNetwork 處理網絡詳細信息請求
func HandleNetwork(req map[string]interface{}) map[string]interface{} {
	// 實現網絡詳細信息收集邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			// 網絡詳細信息
		},
	}
}

// HandleHistory 處理歷史數據管理請求
func HandleHistory(req map[string]interface{}) map[string]interface{} {
	method := req["method"].(string)

	switch method {
	case "GET":
		// 獲取歷史數據
		return handleGetHistory(req)
	case "POST":
		// 生成歷史數據報告
		return handleCreateHistoryReport(req)
	case "DELETE":
		// 刪除歷史數據
		return handleDeleteHistory(req)
	default:
		return map[string]interface{}{
			"status": "error",
			"error":  "Unsupported method",
		}
	}
}

// 獲取歷史數據
func handleGetHistory(req map[string]interface{}) map[string]interface{} {
	// 實現獲取歷史數據邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			// 歷史數據信息
		},
	}
}

// 生成歷史數據報告
func handleCreateHistoryReport(req map[string]interface{}) map[string]interface{} {
	// 實現生成報告邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			// 報告信息
		},
	}
}

// 刪除歷史數據
func handleDeleteHistory(req map[string]interface{}) map[string]interface{} {
	// 實現刪除歷史數據邏輯
	// ...
	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"message": "Historical data deleted successfully",
		},
	}
}

// 主函數，不參與插件運行
func main() {
	fmt.Println("This is a plugin for PanelBase and should not be run directly")
}
```

## 命令配置

### commands.json 文件

`commands.json` 文件位於 `/configs` 目錄下，用於定義系統中可用的命令。

基本結構：

```json
{
  "commands": {
    "user_add": "user/add_user.sh",
    "website_create": "website/create_site.sh",
    "website_delete": "website/delete_site.sh",
    "backup_create": "backup/create_backup.sh",
    "system_update": "system/update.sh"
  }
}
```

配置項說明：

| 欄位       | 說明                                                 |
| ---------- | ---------------------------------------------------- |
| `commands` | 包含所有已安裝命令的對象                             |
| 命令 ID    | 作為鍵，用於在 API 中引用該命令                      |
| 命令路徑   | 作為值，指向命令腳本文件（相對於 `/commands/` 目錄） |

### 命令腳本開發

命令腳本應該存放在 `/commands/{category}/` 目錄下，可以使用 Bash、Python 等腳本語言。命令腳本使用通用參數格式 `*#ARG_N#*`，其中 N 是參數的順序編號（從 1 開始）。

例如：

- `*#ARG_1#*` 表示第一個參數
- `*#ARG_2#*` 表示第二個參數
- 依此類推

腳本開發建議：

1. **參數驗證**：在腳本開始時驗證輸入參數
2. **錯誤處理**：提供有意義的錯誤信息和返回碼
3. **日誌記錄**：記錄關鍵操作和錯誤
4. **幂等性**：命令應該是幂等的，多次執行相同參數的結果應該一致
5. **超時控制**：長時間運行的命令應該有超時機制

### 命令開發實例

下面展示一個網站管理命令的開發實例：

#### 1. 定義命令 JSON 配置

在 `commands.json` 中添加以下配置：

```json
{
  "commands": {
    "website_create": "website/create_site.sh",
    "website_delete": "website/delete_site.sh"
  }
}
```

#### 2. 實現網站創建腳本

在 `/commands/website/create_site.sh` 中實現命令：

```bash
#!/bin/bash

# 檢查是否以root權限運行
if [ "$(id -u)" -ne 0 ]; then
	 echo "{'status': 'error', 'message': 'This script must be run as root'}"
	 exit 1
fi

# 設置變量
DOMAIN="*#ARG_1#*"
ROOT_DIR="*#ARG_2#*"
PHP_VERSION="*#ARG_3#*"
SSL="*#ARG_4#*"
FORCE_WWW="*#ARG_5#*"

# 檢查必要參數
if [ -z "$DOMAIN" ]; then
		echo "{'status': 'error', 'message': 'Domain name is required'}"
		exit 1
fi

# 如果root_dir為空，使用默認值
if [ -z "$ROOT_DIR" ]; then
    ROOT_DIR="/var/www/html/$DOMAIN"
fi

# 如果PHP_VERSION為空，使用默認值
if [ -z "$PHP_VERSION" ]; then
    PHP_VERSION="8.1"
fi

# 如果SSL為空，使用默認值
if [ -z "$SSL" ]; then
    SSL="true"
fi

# 如果FORCE_WWW為空，使用默認值
if [ -z "$FORCE_WWW" ]; then
    FORCE_WWW="false"
fi

# 創建網站目錄
mkdir -p "$ROOT_DIR"

# 設置目錄權限
chown -R www-data:www-data "$ROOT_DIR"
chmod -R 755 "$ROOT_DIR"

# 創建示例index.html
cat > "$ROOT_DIR/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
		<title>Welcome to $DOMAIN</title>
		<style>
				body {
						font-family: Arial, sans-serif;
						line-height: 1.6;
						color: #333;
						max-width: 800px;
						margin: 0 auto;
						padding: 20px;
				}
				h1 {
						color: #2c3e50;
				}
		</style>
</head>
<body>
		<h1>Welcome to $DOMAIN!</h1>
		<p>Your website has been successfully created by PanelBase.</p>
		<p>Replace this file with your own content.</p>
		<p><small>Powered by PanelBase</small></p>
</body>
</html>
EOF

# 創建Nginx配置
NGINX_CONF="/etc/nginx/sites-available/$DOMAIN.conf"

# 基礎配置
cat > "$NGINX_CONF" <<EOF
server {
		listen 80;
		listen [::]:80;
		server_name $DOMAIN;
EOF

# 添加www支持（如果需要）
if [ "$FORCE_WWW" = "true" ]; then
		sed -i "s/server_name $DOMAIN;/server_name $DOMAIN www.$DOMAIN;/" "$NGINX_CONF"

		# 添加強制www重定向
		cat >> "$NGINX_CONF" <<EOF

		# Force redirect to www
		if (\$host = $DOMAIN) {
				return 301 \$scheme://www.$DOMAIN\$request_uri;
		}
EOF
fi

# 完成基本配置
cat >> "$NGINX_CONF" <<EOF
		root $ROOT_DIR;
		index index.html index.htm;

		location / {
				try_files \$uri \$uri/ =404;
		}
EOF

# 添加PHP支持（如果需要）
if [ "$PHP_VERSION" != "none" ]; then
		cat >> "$NGINX_CONF" <<EOF

		# PHP configuration
		location ~ \.php$ {
				include snippets/fastcgi-php.conf;
				fastcgi_pass unix:/var/run/php/php$PHP_VERSION-fpm.sock;
		}

		location ~ /\.ht {
				deny all;
		}
EOF
fi

# 關閉配置文件
cat >> "$NGINX_CONF" <<EOF
}
EOF

# 創建符號鏈接
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/"

# 測試Nginx配置
nginx -t > /dev/null 2>&1
if [ $? -ne 0 ]; then
		echo "{'status': 'error', 'message': 'Nginx configuration test failed'}"
		# 輸出錯誤詳情
		NGINX_ERROR=$(nginx -t 2>&1)
		echo "{'error_details': '${NGINX_ERROR//\'/\\\'}'}"
		exit 1
fi

# 重新啟動Nginx
systemctl reload nginx

# 如果啟用SSL，配置Let's Encrypt
if [ "$SSL" = "true" ]; then
		# 使用Certbot獲取SSL證書
		certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --email admin@"$DOMAIN" --redirect

		if [ $? -ne 0 ]; then
				echo "{'status': 'warning', 'message': 'Website created, but SSL setup failed'}"
				exit 0
		fi
fi

# 輸出成功信息
echo "{'status': 'success', 'message': 'Website created successfully', 'data': {'domain': '$DOMAIN', 'root_dir': '$ROOT_DIR', 'ssl_enabled': $SSL}}"
exit 0
```

## 版本檢查與更新機制

### PATCH 方法的版本驗證

PATCH 請求用於版本檢查和更新：

1. 客戶端發送 PATCH 請求，僅包含資源 ID：`{"id":"target_id"}`
2. 服務器檢查本地版本與源鏈接版本
3. 如果有更新，僅替換變化的部分，保留本地目錄中的文件

這種機制適用於主題、插件和命令的更新。

### 更新策略與優化

PanelBase 使用以下更新策略：

1. **選擇性更新**：僅下載和替換變更的文件
2. **版本比較**：使用語義化版本號進行比較
3. **備份機制**：更新前備份現有文件
4. **回滾能力**：更新失敗時能夠回滾到先前版本
5. **變更日誌**：記錄每次更新的變化

更新過程保留現有目錄結構和文件，只替換有變化的部分，確保用戶的自定義設置不會丟失。

### 更新流程示例

以下是使用 PATCH 方法更新主題的完整流程示例：

1. **檢查更新**：客戶端發送 PATCH 請求，僅包含主題 ID

```bash
curl -X PATCH -H "Authorization: Bearer YOUR_TOKEN" -H "Content-Type: application/json" -d '{"id":"custom_theme"}' http://your-server-ip:8080/api/v1/theme
```

2. **服務器端處理流程**：

```go
// 處理PATCH請求 - 主題更新
func (h *ThemeHandler) updateTheme(c *gin.Context, themeID string) {
		// 獲取當前主題信息
		themeInfo, exists := h.configService.ThemesConfig.Themes[themeID]
		if !exists {
				c.JSON(http.StatusNotFound, gin.H{
						"status": "error",
						"error":  "Theme not found",
				})
				return
		}

		// 檢查源連接
		sourceLink := themeInfo.SourceLink
		if sourceLink == "" {
				c.JSON(http.StatusBadRequest, gin.H{
						"status": "error",
						"error":  "Theme has no source link for updates",
				})
				return
		}

		// 獲取遠程主題信息
		remoteThemeInfo, err := fetchRemoteThemeInfo(sourceLink)
		if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
						"status": "error",
						"error":  "Failed to fetch theme information: " + err.Error(),
				})
				return
		}

		// 比較版本
		currentVersion := themeInfo.Version
		remoteVersion := remoteThemeInfo.Version

		if !isNewerVersion(currentVersion, remoteVersion) {
				c.JSON(http.StatusOK, gin.H{
						"status":  "success",
						"message": "Theme is already up to date",
						"data": gin.H{
								"current_version": currentVersion,
								"remote_version":  remoteVersion,
						},
				})
				return
		}

		// 創建備份
		backupDir := fmt.Sprintf("./backups/themes/%s_%s", themeID, time.Now().Format("20060102_150405"))
		err = backupTheme(themeID, backupDir)
		if err != nil {
				log.Printf("Warning: Failed to create backup: %v", err)
				// 繼續執行，即使備份失敗
		}

		// 下載並更新變更的文件
		updatedFiles, err := updateThemeFiles(themeID, remoteThemeInfo)
		if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
						"status": "error",
						"error":  "Failed to update theme: " + err.Error(),
				})
				return
		}

		// 更新版本信息
		themeInfo.Version = remoteVersion
		h.configService.ThemesConfig.Themes[themeID] = themeInfo

		// 保存配置
		err = h.configService.SaveThemesConfig()
		if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
						"status": "error",
						"error":  "Failed to save configuration: " + err.Error(),
				})
				return
		}

		c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"message": "Theme updated successfully",
				"data": gin.H{
						"theme_id":        themeID,
						"previous_version": currentVersion,
						"new_version":      remoteVersion,
						"updated_files":    updatedFiles,
				},
		})
}

// 檢查是否為更新版本
func isNewerVersion(current, remote string) bool {
		// 使用語義化版本比較
		currentVersion, err := semver.Parse(current)
		if err != nil {
				return remote != current // 如果無法解析，直接比較字符串
		}

		remoteVersion, err := semver.Parse(remote)
		if err != nil {
				return remote != current
		}

		return remoteVersion.GT(currentVersion)
}

// 更新主題文件
func updateThemeFiles(themeID string, remoteInfo ThemeInfo) ([]string, error) {
		themeDir := fmt.Sprintf("./web/%s", remoteInfo.Directory)
		var updatedFiles []string

		// 確保目錄存在
		if err := os.MkdirAll(themeDir, 0755); err != nil {
				return nil, err
		}

		// 遍歷遠程結構
		for filePath, fileURL := range remoteInfo.Structure {
				targetPath := fmt.Sprintf("%s/%s", themeDir, filePath)

				// 檢查目錄是否存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
						return updatedFiles, err
				}

				// 下載文件
				if err := downloadFile(fileURL, targetPath); err == nil {
						updatedFiles = append(updatedFiles, filePath)
				} else {
						log.Printf("Warning: Failed to download file %s: %v", filePath, err)
				}
		}

		return updatedFiles, nil
}
```

3. **客戶端響應示例**：

```json
{
  "status": "success",
  "message": "Theme updated successfully",
  "data": {
    "theme_id": "custom_theme",
    "previous_version": "1.0.0",
    "new_version": "1.1.0",
    "updated_files": ["css/style.css", "js/script.js", "dashboard.html"]
  }
}
```

這個更新流程展示了 PanelBase 的版本檢查和選擇性更新機制，確保只有變更的文件被替換，從而最大限度地保留用戶的自定義設置。
