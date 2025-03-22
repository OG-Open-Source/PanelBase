# PanelBase

PanelBase 是一个基于 Go 1.19 和 Echo 框架构建的网络应用程序框架，专为提供具有可定制入口点的主题化网络内容而设计。

## 特性

- 支持 Windows、Linux 和 macOS 的跨平台运行
- 通过 TOML 配置文件的服务器设置
- 通过 JSON 配置的主题化网络界面
- 基于 JWT 的身份验证
- 结构化日志记录，支持文件轮替
- RESTful API 设计
- 首次运行时自动配置：
  - 自动创建所需目录
  - 在 1024-49151 范围内自动分配可用端口
  - 自动生成 12 位随机入口代码
  - 自动生成 512 位 JWT 密钥

## 安装

### 前提条件

- Go 1.19 或更高版本
- Git

### 开始使用

1. 克隆仓库:

   ```
   git clone https://github.com/OG-Open-Source/PanelBase.git
   cd PanelBase
   ```

2. 安装依赖项:

   ```
   go mod tidy
   ```

3. 构建应用程序:

   ```
   go build -o panelbase ./cmd/panelbase/
   ```

4. 运行应用程序:
   ```
   ./panelbase
   ```

## 配置

PanelBase 的设计允许无需预先配置。首次运行时，它将自动:

1. 创建 `configs` 目录
2. 生成包含随机端口、入口代码和 JWT 密钥的 `config.toml`
3. 创建默认主题和必要的文件结构

### 服务器配置 (config.toml)

配置文件位于 `configs/config.toml`:

```toml
[server]
host = "0.0.0.0"
port = 37415  # 自动分配，范围为 1024-49151
entry = "R0wrGu2shcHU"  # 自动生成 12 位随机码

[security]
jwt_secret = "your_jwt_secret"  # 自动生成 512 位随机密钥
jwt_expire_hours = 24

[logging]
level = "info"
file = "logs/panelbase.log"
max_size = 10
max_backups = 5
max_age = 30
```

- `server.entry` 定义了访问 Web 界面的入口路径

### 主题配置 (theme.json)

主题配置位于 `configs/theme.json`:

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
      "directory": "default",
      "structure": {
        "index.html": "web/default/index.html",
        "style.css": "web/default/style.css",
        "script.js": "web/default/script.js"
      }
    }
  }
}
```

## API 端点

### Web 界面

- `GET /{entry}/` - 根据配置的主题提供 Web 界面
- `GET /{entry}/theme/info` - 返回关于当前主题的元数据

### 主题管理

- `GET /{entry}/theme/download?url=<theme_url>` - 下载指定 URL 的主题文件
  - URL 必须是 JSON 格式文件（以 .json 结尾）
  - 会验证 URL 格式及 JSON 内容是否有效
  - 主题文件将被保存到服务器上
  - 返回下载成功信息和保存路径

- `GET /{entry}/theme/metadata?url=<theme_url>` - 检查指定 URL 的主题元数据
  - URL 必须是 JSON 格式文件（以 .json 结尾）
  - 会验证 URL 格式及 JSON 内容是否有效
  - 返回完整的主题元数据 JSON 内容

### 系统端点

- `GET /health` - 用于监控的健康检查端点
  - 返回: `{"status":"ok"}` 和 200 状态码

### 主题 API 响应格式

主题信息端点返回以下 JSON 结构:

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

## 开发

### 项目结构

```
├── cmd/
│   └── panelbase/      # 主应用程序入口点
├── configs/            # 配置文件（自动生成）
├── internal/
│   ├── api/            # API 服务器实现
│   ├── handlers/       # 请求处理程序
│   └── middleware/     # 自定义中间件
├── pkg/
│   ├── config/         # 配置管理
│   ├── logger/         # 日志实用工具
│   ├── theme/          # 主题管理
│   └── utils/          # 通用工具函数
├── web/                # 主题文件（自动生成）
│   └── default/        # 默认主题
└── logs/               # 应用程序日志（自动生成）
```

## 许可证

本项目根据 MIT 许可证授权 - 有关详细信息，请参阅 LICENSE 文件。

## 更新日志

### v0.1.3 (2025-03-23)
- 优化 API 路径：将主题元数据检查路径改为 `/theme/metadata`
- 添加主题下载 API 端点，支持从 URL 下载主题文件
- 添加主题元数据检查 API 端点，支持检查主题文件有效性
- 修复了服务器关闭时错误处理的 bug
- 改进了服务器的优雅关闭机制
- 使用 context 控制关闭超时
- 优化主题信息端点，现返回完整主题内容

### v0.1.2 (2025-03-22)

- 主程序路径更改为 cmd/panelbase/main.go
- 添加自动配置功能：在缺少配置时生成配置
- 实现自动端口分配 (1024-49151 范围内)
- 添加自动生成随机入口代码 (12 位)
- 添加自动生成 JWT 密钥 (512 位)
- 锁定 Go 版本为 1.19 以确保兼容性

### v0.1.1 (2025-03-22)

- 更新主题配置以使用本地文件
- 修复主题加载中的错误处理
- 改进文档和构建脚本
- 增强了包含响应格式的 API 文档
- 添加跨平台构建脚本（Windows、Linux、macOS）
- 在 Windows 10.0.26100 上成功测试

### v0.1.0 (2023-03-22)

- 初始项目结构
- 基于 Echo 框架的基本 API 服务器
- 服务器和主题的配置管理
- 默认主题实现
- 基于主题配置的 Web 内容提供
- 结构化日志记录，支持文件轮替
