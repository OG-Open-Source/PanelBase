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