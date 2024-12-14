# PanelBase
一個基於 Lighttpd 的輕量級面板基礎框架，支援自定義前端介面和 CGI 擴展。

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
PanelBase 是一個輕量級的面板基礎框架，基於 Lighttpd 構建。它提供了基礎的認證系統和 CGI 支援，允許使用者根據需求自行擴展功能。無論是通過 Markdown 快速構建介面，還是開發完整的 HTML 前端，都能輕鬆實現。

## 特性
- 輕量級設計，易於部署和使用
- 支援主流 Linux 發行版（Debian、Ubuntu、CentOS 等）
- 單一管理員帳號系統
- CGI 介面支援，方便功能擴展
- 支援 Markdown 轉 HTML（可選）
- 多語言支援
- 完整的操作日誌記錄
- 彈性的前端介面選擇（Markdown 或自定義 HTML）

## 安裝
### 系統要求
- 支援 Lighttpd 的 Linux 系統
- Bash Shell 環境

### 安裝步驟
```bash
# 克隆倉庫
git clone https://github.com/username/PanelBase.git

# 進入目錄
cd PanelBase

# 執行安裝腳本
bash install.sh
```

## 使用方法
1. 安裝完成後，訪問 http://your-server-ip:port
2. 使用預設帳號登入（安裝時設定）
3. 根據需求修改配置和擴展功能

### CGI 開發
在 `src/cgi-bin` 目錄下，您可以使用多種程式語言開發 CGI 腳本：

#### Bash 腳本 (.sh)
```bash
#!/bin/bash
echo "Content-type: application/json"
echo ""
echo '{"message": "Hello from Bash!"}'
```

#### Python 腳本 (.py)
```python
#!/usr/bin/env python3
print("Content-Type: application/json")
print()
print('{"message": "Hello from Python!"}')
```

#### Perl 腳本 (.pl)
```perl
#!/usr/bin/perl
print "Content-type: application/json\n\n";
print '{"message": "Hello from Perl!"}';
```

#### Ruby 腳本 (.rb)
```ruby
#!/usr/bin/ruby
puts "Content-type: application/json\n\n"
puts '{"message": "Hello from Ruby!"}'
```

所有腳本都需要：
1. 設置正確的檔案權限（chmod 755）
2. 包含正確的 shebang 行（#!/path/to/interpreter）
3. 設置正確的 Content-Type 標頭

支援的腳本類型：
- `.sh`：Bash 腳本
- `.py`：Python 腳本
- `.pl`：Perl 腳本
- `.rb`：Ruby 腳本
- `.cgi`：通用 CGI 腳本

## 示例
### Markdown 轉換示例
```markdown
# 我的面板
## 功能列表
- 功能 1
- 功能 2
```

## 配置
主要配置文件位於 `src/config` 目錄：

```ini
# panel.conf
PORT=8080
LOG_LEVEL=info
```

## 常見問題
**Q：如何修改管理員密碼？**
A：通過面板的設定介面或直接修改配置文件。

**Q：如何添加新的 CGI 腳本？**
A：將腳本放入 `src/cgi-bin` 目錄，確保具有執行權限。

## 貢獻指南
1. Fork 專案
2. 創建功能分支
3. 提交更改
4. 發起 Pull Request

## 許可證
本專案採用 MIT 許可證。
