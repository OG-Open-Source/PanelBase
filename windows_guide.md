# Windows 系统下的 PanelBase API 使用指南

## 构建与运行

1. **构建应用程序**

   打开命令提示符或 PowerShell，导航到项目目录，执行以下命令：

   ```powershell
   go build -o bin/panelbase.exe ./cmd/panelbase/
   ```

2. **运行应用程序**

   ```powershell
   .\bin\panelbase.exe
   ```

   首次运行时，应用程序将：

   - 自动创建 `configs` 目录
   - 生成 `config.toml` 配置文件，包含：
     - 随机端口 (1024-49151 范围内)
     - 随机入口点 (12 位字符串)
     - 随机 JWT 密钥
   - 创建默认主题

   在控制台中应该能看到类似以下的输出：

   ```
   http server started on [::]:12088
   ```

3. **查看配置信息**

   ```powershell
   type .\configs\config.toml
   ```

   记下端口号和入口点值，这些将用于访问 API。

## API 端点使用方法

假设应用程序运行在端口 12088，入口点是 2d3f5527a780，您可以访问以下 API：

### 1. 健康检查端点

**请求：**

```powershell
Invoke-WebRequest -Uri "http://localhost:12088/health"
```

**预期响应：**

```json
{ "status": "ok" }
```

### 2. 主题信息端点

**请求：**

```powershell
Invoke-WebRequest -Uri "http://localhost:12088/2d3f5527a780/theme/info"
```

**预期响应：**

```json
{
  "name": "Default Theme",
  "authors": "PanelBase Team",
  "version": "1.0.0",
  "description": "Default theme for PanelBase",
  "source_link": "https://github.com/OG-Open-Source/PanelBase",
  "directory": "default",
  "structure": {
    "index.html": "web/default/index.html",
    "style.css": "web/default/style.css",
    "script.js": "web/default/script.js"
  }
}
```

### 3. 主题下载端点

**请求：**

```powershell
Invoke-WebRequest -Uri "http://localhost:12088/2d3f5527a780/theme/download?url=https://example.com/mytheme.json"
```

**预期响应：**

```json
{
  "status": "success",
  "message": "Theme downloaded successfully",
  "file_path": "themes/temp/mytheme.json",
  "theme_name": "My Custom Theme"
}
```

### 4. 主题元数据检查端点

**请求：**

```powershell
Invoke-WebRequest -Uri "http://localhost:12088/2d3f5527a780/theme/metadata?url=https://example.com/mytheme.json"
```

**预期响应：**

```json
{
  "name": "My Custom Theme",
  "authors": "Theme Author",
  "version": "1.0.0",
  "description": "A custom theme for PanelBase",
  "source_link": "https://github.com/author/mytheme",
  "directory": "custom",
  "structure": {
    "index.html": "web/custom/index.html",
    "style.css": "web/custom/style.css",
    "script.js": "web/custom/script.js"
  }
}
```

### 5. 访问 Web 界面

在浏览器中打开以下 URL：

```
http://localhost:12088/2d3f5527a780/
```

这将加载默认主题的 Web 界面。

## 使用 curl（如果已安装）

如果您的 Windows 系统上安装了 curl，也可以使用 curl 命令访问 API：

### 健康检查端点

```
curl http://localhost:12088/health
```

### 主题信息端点

```
curl http://localhost:12088/2d3f5527a780/theme/info
```

## 常见问题解决

1. **端口被占用**

   如果默认端口被占用，您可以修改 `configs/config.toml` 文件中的端口号，然后重启应用程序。

2. **应用程序无法启动**

   检查是否有另一个 panelbase.exe 实例正在运行：

   ```powershell
   tasklist | findstr panelbase
   ```

   如果有，可以使用以下命令终止它：

   ```powershell
   taskkill /F /IM panelbase.exe
   ```

3. **访问 API 出现错误**

   确保使用正确的端口号和入口点。这些值在每次重新生成配置时都会变化。
