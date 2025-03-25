# PanelBase

PanelBase 是一个跨平台的、可自定义的面板系统，允许用户通过网页界面控制和监控系统。它由 Go 语言开发，支持 Windows、macOS 和 Linux，提供了一套模块化的框架来创建管理界面。

## 特性

- 跨平台支持（Windows、macOS、Linux）
- 基于 Go 1.19 开发
- 模块化设计，易于扩展
- 自定义主题支持
- 基于角色的访问控制
- RESTful API
- 响应式 Web 界面

## 目录结构

```
PanelBase/
├── cmd/panelbase/         # 程序入口点
├── internal/              # 内部包
│   ├── app/               # 核心应用程序
│   │   ├── api/           # API 接口
│   │   ├── auth/          # 认证相关
│   │   ├── config/        # 配置处理
│   │   ├── core/          # 核心功能
│   │   ├── handler/       # HTTP 处理器
│   │   ├── model/         # 数据模型
│   │   └── service/       # 业务逻辑
│   └── pkg/               # 内部使用的工具包
├── pkg/                   # 可导出的公共库
├── configs/               # 配置文件目录
├── web/                   # Web 资源
│   └── themes/            # 主题目录
├── go.mod                 # Go 模块声明
└── README.md              # 项目说明
```

## 安装和使用

### 从源码构建

1. 克隆仓库：

```bash
git clone https://github.com/OG-Open-Source/PanelBase.git
cd PanelBase
```

2. 构建项目：

```bash
go build -o panelbase ./cmd/panelbase
```

3. 运行：

```bash
./panelbase
```

默认配置文件位于 `configs/config.toml`，如需使用自定义配置，可通过 `-config` 参数指定：

```bash
./panelbase -config /path/to/config.toml
```

### 配置

PanelBase 使用以下配置文件：

- `config.toml`: 主要配置文件，包含日志和服务器设置
- `users.json`: 用户数据和认证信息
- `theme.json`: 主题配置
- `routes.json`: 路由配置，定义 API 路由与处理函数的映射

## 开发

### 依赖

- Go 1.19 或更高版本
- 现代化浏览器（Chrome、Firefox、Safari、Edge）

### 构建开发版本

```bash
go build -tags dev -o panelbase-dev ./cmd/panelbase
```

## 更新日志

### 2023-03-25

- 初始化项目结构
- 实现基本的服务器框架和路由
- 添加用户认证系统
- 创建默认主题和模板
- 添加配置文件处理功能

## 贡献指南

欢迎提交 Pull Requests 和 Issues 来改进 PanelBase。请确保您的代码符合项目的代码风格和最佳实践。

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。 