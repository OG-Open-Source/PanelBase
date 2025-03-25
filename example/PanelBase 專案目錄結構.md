# **PanelBase 專案目錄結構**

## **1\. 專案概述**

本文件描述了 PanelBase 專案的目錄結構。PanelBase 是一個跨平台的應用程式，旨在提供一個可自訂的面板系統，允許使用者透過網頁介面控制和監控系統。

## **2\. 目錄結構**

PanelBase/  
├── cmd/  
│ └── panelbase/ \# 專案的主要應用程式  
│ └── main.go \# 應用程式的入口點  
├── internal/  
│ ├── app/ \# 應用程式的核心邏輯  
│ │ ├── api/ \# API 相關程式碼  
│ │ │ ├── auth/ \# 身份驗證相關  
│ │ │ │ └── auth.go  
│ │ │ └── routes.go \# API 路由定義  
│ │ ├── config/ \# 應用程式配置  
│ │ │ └── config.go  
│ │ ├── core/ \# 核心功能模組  
│ │ │ ├── module/ \# 模組管理  
│ │ │ │ └── module.go  
│ │ │ ├── task/ \# 任務管理  
│ │ │ │ └── task.go  
│ │ │ ├── permission/ \# 權限管理  
│ │ │ │ └── permission.go  
│ │ │ └── router/ \# 路由文件解析與執行  
│ │ │ └── router.go  
│ │ ├── handler/ \# HTTP 請求處理  
│ │ │ └── handler.go  
│ │ ├── model/ \# 資料模型  
│ │ │ └── model.go  
│ │ └── service/ \# 業務邏輯服務  
│ │ └── service.go  
│ ├── pkg/ \# 可重用的函式庫  
│ │ └── utils/ \# 通用工具函式  
│ │ └── utils.go  
│ └── web/ \# Web 相關資源  
│ ├── themes/ \# 主題目錄  
│ │ ├── default/ \# 預設主題  
│ │ │ ├── assets/ \# 靜態資源（CSS、JavaScript）  
│ │ │ │ └── ...  
│ │ │ └── templates/ \# HTML 模板  
│ │ │ └── index.html  
│ │ └── ... \# 其他主題  
│ └── web.go \# Web 伺服器設定  
├── configs/ \# 應用程式配置檔案  
│ ├── config.toml \# 應用程式主要配置檔案 (TOML 格式)  
│ ├── users.json \# 使用者資料 (JSON 格式)  
│ ├── themes.json \# 主題資料 (JSON 格式)  
│ ├── routes.json \# 路由資料 (JSON 格式)  
├── Dockerfile \# Dockerfile  
├── go.mod \# Go 模組定義  
├── go.sum \# Go 模組依賴  
└── README.md

## **3\. 目錄結構說明**

- **cmd/panelbase/**：
  - 包含應用程式的主要進入點 main.go。
- **internal/app/**：
  - 包含應用程式的核心邏輯。
  - **api/**：
    - 包含 API 相關程式碼，包括身份驗證和路由定義。
  - **config/**：
    - 包含應用程式的配置管理。
  - **core/**：
    - 包含模組、任務、權限，以及路由文件解析與執行的程式碼。
  - **handler/**：
    - 包含 HTTP 請求處理程式。
  - **model/**：
    - 包含資料模型定義。
  - **service/**：
    - 包含業務邏輯服務。
- **internal/pkg/utils/**：
  - 包含可重用的通用工具函式。
- **internal/web/**：
  - 包含 Web 相關資源，包括主題目錄和 Web 伺服器設定。
- **configs/**：
  - 包含應用程式的配置檔案。
    - config.toml: 應用程式主要配置檔案，使用 TOML 格式。
    - users.json: 使用者資料，包含使用者 ID、使用者名稱、密碼雜湊、角色等。
    - themes.json: 主題資料，包含主題 ID、名稱、作者、版本、描述、目錄和檔案結構。
    - routes.json: 路由資料，定義程式碼與路由的映射。
- **Dockerfile**：
  - 包含 Dockerfile，用於 Docker 容器化。
- **go.mod** 和 **go.sum**：
  - Go 模組定義和依賴管理。
- **README.md**：
  - 專案說明文件。

## **4\. 設計考量**

- **模組化設計：** 將核心功能拆分為模組、任務、權限和路由，提高程式碼的可維護性和可擴展性。
- **Web 資源管理：** 將 Web 資源放置在 internal/web/ 目錄下，方便管理和部署。
- **API 設計：** 將 API 相關程式碼放置在 internal/app/api/ 目錄下，方便管理和擴展。
- **配置管理：**
  - 使用 internal/app/config/ 目錄管理應用程式配置。
  - 採用 config.toml 作為主要配置檔案，支援巢狀結構和易於閱讀的格式。
  - 使用 JSON 檔案儲存使用者、主題和路由資料，方便讀取和修改。
- **Docker 支援:** 提供 Dockerfile，方便將應用程式打包成 Docker 映像檔。

## **5\. 結論**

這個更新後的目錄結構，能夠更完善地管理 PanelBase 專案的程式碼和資源，並符合您提供的配置檔案格式和需求。
