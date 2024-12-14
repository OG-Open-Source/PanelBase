# 管理面板

## 系統資訊
- 系統狀態：<span id="system-status">載入中...</span>
- CPU 使用率：<span id="cpu-usage">載入中...</span>
- 記憶體使用率：<span id="memory-usage">載入中...</span>

## 功能選單
- [系統設定](#system-settings)
- [帳號管理](#account-management)
- [日誌查看](#log-viewer)

## 系統設定 {#system-settings}
### 基本設定
- 面板端口：<input type="text" id="panel-port" value="8080">
- 日誌等級：
  <select id="log-level">
    <option value="debug">Debug</option>
    <option value="info">Info</option>
    <option value="warn">Warning</option>
    <option value="error">Error</option>
  </select>

### 安全設定
- 啟用 HTTPS：<input type="checkbox" id="enable-https">
- 允許的 IP：<input type="text" id="allowed-ips" placeholder="例如：192.168.1.0/24">

## 帳號管理 {#account-management}
### 修改密碼
- 當前密碼：<input type="password" id="current-password">
- 新密碼：<input type="password" id="new-password">
- 確認密碼：<input type="password" id="confirm-password">
<button onclick="changePassword()">更新密碼</button>

## 日誌查看 {#log-viewer}
<div id="log-content" style="height: 300px; overflow-y: scroll; background: #f5f5f5; padding: 10px;">
日誌內容將顯示在此處...
</div>

<style>
input, select {
    margin: 5px;
    padding: 5px;
    border: 1px solid #ddd;
    border-radius: 3px;
}

button {
    background: #4CAF50;
    color: white;
    padding: 8px 15px;
    border: none;
    border-radius: 3px;
    cursor: pointer;
}

button:hover {
    background: #45a049;
}
</style>

<script>
// 這裡可以添加您的 JavaScript 代碼
function changePassword() {
    // 實現密碼更改邏輯
}

// 定期更新系統資訊
setInterval(() => {
    // 實現系統資訊更新邏輯
}, 5000);
</script> 