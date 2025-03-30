# API 回應格式規範示例

本文檔提供 PanelBase API 回應格式的規範和示例，所有 API 路由（包括插件）都必須遵循此格式。

## 基本格式

所有 API 回應必須使用以下統一格式：

```json
{
  "status": "success", // 或 "failure"
  "message": "操作結果說明",
  "data": {
    // 可選，操作結果相關的數據
  }
}
```

## 成功回應示例

### 獲取資源列表

請求：

```
GET /api/v1/themes
```

回應：

```json
{
  "status": "success",
  "message": "成功獲取主題列表",
  "data": {
    "themes": [
      {
        "id": "default",
        "name": "默認主題",
        "version": "1.0.0",
        "is_current": true
      },
      {
        "id": "dark",
        "name": "暗黑主題",
        "version": "1.2.1",
        "is_current": false
      }
    ],
    "total": 2
  }
}
```

### 創建資源

請求：

```
POST /api/v1/plugins
```

請求體：

```json
{
  "url": "https://example.com/plugins/stats.json"
}
```

回應：

```json
{
  "status": "success",
  "message": "插件安裝成功",
  "data": {
    "plugin_id": "stats_plugin",
    "name": "系統統計插件",
    "version": "1.0.0",
    "installed_at": "2024-07-08T15:30:45Z"
  }
}
```

### 無數據操作

請求：

```
DELETE /api/v1/plugins
```

請求體：

```json
{
  "id": "unused_plugin"
}
```

回應：

```json
{
  "status": "success",
  "message": "插件已成功刪除",
  "data": {}
}
```

### 批量執行命令

請求：

```
POST /api/v1/execute
```

請求體：

```json
{
  "commands": [
    {
      "command": "website_create",
      "args": ["example.com", "/var/www/html"]
    },
    {
      "command": "database_create",
      "args": ["example_db", "utf8mb4"]
    },
    {
      "command": "firewall_allow",
      "args": ["80", "443"]
    }
  ]
}
```

回應：

```json
{
  "status": "success",
  "message": "所有命令執行成功",
  "data": {
    "results": [
      {
        "command": "website_create",
        "status": "success",
        "output": "網站 example.com 創建成功"
      },
      {
        "command": "database_create",
        "status": "success",
        "output": "數據庫 example_db 創建成功"
      },
      {
        "command": "firewall_allow",
        "status": "success",
        "output": "已允許端口 80, 443"
      }
    ],
    "execution_time": "2.5s"
  }
}
```

## 失敗回應示例

### 資源不存在

請求：

```
GET /api/v1/plugins
```

請求體：

```json
{
  "id": "nonexistent"
}
```

回應：

```json
{
  "status": "failure",
  "message": "找不到指定的插件",
  "data": {
    "error_code": "RESOURCE_NOT_FOUND",
    "plugin_id": "nonexistent"
  }
}
```

### 參數錯誤

請求：

```
POST /api/v1/commands
```

請求體：

```json
{
  "wrong_parameter": "value"
}
```

回應：

```json
{
  "status": "failure",
  "message": "無效的請求參數",
  "data": {
    "error_code": "INVALID_PARAMETERS",
    "details": "缺少必要參數 'url'",
    "provided_parameters": ["wrong_parameter"]
  }
}
```

### 權限不足

請求：

```
PUT /api/v1/users
```

請求體：

```json
{
  "id": "admin",
  "role": "super_admin"
}
```

回應：

```json
{
  "status": "failure",
  "message": "權限不足",
  "data": {
    "error_code": "INSUFFICIENT_PERMISSIONS",
    "required_role": "admin",
    "current_role": "user"
  }
}
```

### 命令執行錯誤

請求：

```
POST /api/v1/execute
```

請求體：

```json
{
  "commands": [
    {
      "command": "website_create",
      "args": ["example.com", "/var/www/html"]
    },
    {
      "command": "database_create",
      "args": ["invalid@db", "utf8mb4"]
    }
  ]
}
```

回應：

```json
{
  "status": "failure",
  "message": "部分命令執行失敗",
  "data": {
    "error_code": "COMMAND_EXECUTION_ERROR",
    "results": [
      {
        "command": "website_create",
        "status": "success",
        "output": "網站 example.com 創建成功"
      },
      {
        "command": "database_create",
        "status": "failure",
        "error": "數據庫名稱包含無效字符",
        "index": 1
      }
    ],
    "execution_stopped_at": 1
  }
}
```

## 實現指南

### Go 語言實現

使用 handlers 包中提供的工具函數：

```go
// 成功回應
func MyHandler(c *gin.Context) {
    data := map[string]interface{}{
        "items": items,
        "total": len(items),
    }
    handlers.SendSuccessResponse(c, "操作成功", data)
}

// 錯誤回應
func ErrorHandler(c *gin.Context) {
    errorDetails := map[string]interface{}{
        "error_code": "VALIDATION_ERROR",
        "field": "email",
        "reason": "格式不正確",
    }
    handlers.SendErrorResponse(c, http.StatusBadRequest, "請求驗證失敗", errorDetails)
}
```

### 插件實現

在插件中返回符合規範的響應：

```go
func HandleRequest() *models.PluginAPIResponse {
    return &models.PluginAPIResponse{
        Status:  "success",
        Message: "數據處理成功",
        Data: map[string]interface{}{
            "result": "處理結果",
            "count": 42,
        },
    }
}
```
