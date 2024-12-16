# PanelBase

基於 lighttpd CGI 的輕量級網頁管理面板。

---

## 目錄
- [簡介](#簡介)
- [特性](#特性)
- [安裝](#安裝)
- [使用方法](#使用方法)
- [示例](#示例)
- [配置](#配置)
- [常見問題](#常見問題)
- [貢獻指南](#貢獻指南)
- [許可證](#許可證)
- [API 文檔](#api-文檔)

---

## 簡介
PanelBase 是一個基於 lighttpd 和 CGI 技術的輕量級網頁管理面板，提供簡單易用的網頁介面來管理您的系統。

## 特性
- 輕量級設計，資源佔用少
- 基於 lighttpd 的高效能 Web 服務
- CGI 腳本支援，易於擴展
- 簡潔的使用者介面
- 安全的認證機制
- 多平台支援（Linux、Windows）

## 安裝
執行以下命令即可快速安裝：

```bash
curl -sSLO "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/install.sh"
chmod +x install.sh
sudo ./install.sh
```

### 系統需求
- lighttpd 1.4+
- Bash（用於安裝腳本）
- curl（用於下載）

## 使用方法
1. 安裝完成後，訪問 `http://your-ip:8080`
2. 使用您在安裝時設定的帳號密碼登入
3. 開始使用面板功能

## 示例
安裝完成後，您可以：
- 通過網頁介面管理系統服務
- 查看系統狀態
- 管理檔案
- 配置系統設定

## 配置
配置文件位於 `config/lighttpd.conf`：

```conf
# 基本配置示例
server.port = 8080
server.document-root = "/var/www"
cgi.assign = ( ".cgi" => "" )
```

## 常見問題
**Q：如何修改預設埠號？**<br>
A：編輯 config/lighttpd.conf 文件中的 server.port 值

**Q：忘記管理員密碼怎麼辦？**<br>
A：執行 reset_password.sh 腳本重置密碼

## 貢獻指南
1. Fork 專案
2. 創建功能分支
3. 提交更改
4. 發起 Pull Request

## 許可證
本專案採用 MIT 許可證。 

## API 文檔

所有 API 都支援使用 curl 呼叫。以下範例中的 `$TOKEN` 為登入後獲得的認證令牌。

### 認證相關

#### 登入
```http
POST /cgi-bin/auth.cgi?action=login
Content-Type: application/x-www-form-urlencoded

username=admin&password=password
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/cgi-bin/auth.cgi?action=login" \
     -d "username=admin&password=password" \
     -c cookies.txt \
     -sSL
```

#### 登出
```http
POST /cgi-bin/auth.cgi?action=logout
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/cgi-bin/auth.cgi?action=logout" \
     -b cookies.txt \
     -sSL
```

#### 獲取當前用戶
```http
GET /api/panel/username
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/panel/username" \
     -b cookies.txt \
     -sSL
```

### 面板相關

#### 獲取系統資訊
```http
GET /api/panel/system_info
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/panel/system_info" \
     -b cookies.txt \
     -sSL
```

### 系統管理

#### 獲取系統資訊
```http
GET /api/system/info
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/system/info" \
     -b cookies.txt \
     -sSL
```

### 服務管理

#### 獲取服務狀態
```http
GET /api/service/status/nginx
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/service/status/nginx" \
     -b cookies.txt \
     -sSL
```

#### 啟動服務
```http
POST /api/service/start/nginx
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/service/start/nginx" \
     -b cookies.txt \
     -sSL
```

### 進程管理

#### 列出進程
```http
GET /api/process/list
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/process/list" \
     -b cookies.txt \
     -sSL
```

#### 結束進程
```http
POST /api/process/kill/1234
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/process/kill/1234" \
     -b cookies.txt \
     -sSL
```

### 用戶管理

#### 列出用戶
```http
GET /api/user/list
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/user/list" \
     -b cookies.txt \
     -sSL
```

#### 添加用戶
```http
POST /api/user/add
Content-Type: application/x-www-form-urlencoded

username=newuser&password=newpass
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/user/add" \
     -d "username=newuser&password=newpass" \
     -b cookies.txt \
     -sSL
```

#### 修改密碼
```http
POST /api/user/password
Content-Type: application/x-www-form-urlencoded

username=admin&password=newpass
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/user/password" \
     -d "username=admin&password=newpass" \
     -b cookies.txt \
     -sSL
```

### 網絡工具

#### Ping 測試
```http
GET /api/network/ping/8.8.8.8
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/network/ping/8.8.8.8" \
     -b cookies.txt \
     -sSL
```

### 檔案系統

#### 列出檔案系統資訊
```http
GET /api/fs/list
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/fs/list" \
     -b cookies.txt \
     -sSL
```

### 日誌查看

#### 系統日誌
```http
GET /api/logs/system
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/logs/system" \
     -b cookies.txt \
     -sSL
```

### 防火牆管理

#### 添加規則
```http
POST /api/firewall/allow/80
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/firewall/allow/80" \
     -b cookies.txt \
     -sSL
```

### 系統更新

#### 執行更新
```http
POST /api/system/update
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/system/update" \
     -b cookies.txt \
     -sSL
```

### 備份管理

#### 創建備份
```http
POST /api/backup/create/mybackup
Content-Type: application/x-www-form-urlencoded

path=/path/to/backup
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/backup/create/mybackup" \
     -d "path=/path/to/backup" \
     -b cookies.txt \
     -sSL
```

### 資料庫管理

#### MySQL 狀態
```http
GET /api/mysql/status
```

使用 curl：
```bash
curl -X GET "http://localhost:8080/api/mysql/status" \
     -b cookies.txt \
     -sSL
```

### Docker 管理

#### 容器操作
```http
POST /api/docker/start/container_id
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/docker/start/container_id" \
     -b cookies.txt \
     -sSL
```

### 自定義命令

#### 執行自定義命令
```http
POST /api/custom/shell
Content-Type: application/x-www-form-urlencoded

command=ls -la
```

使用 curl：
```bash
curl -X POST "http://localhost:8080/api/custom/shell" \
     -d "command=ls -la" \
     -b cookies.txt \
     -sSL
```