#!/bin/bash

# 設置 HTTP 響應頭
echo "Content-Type: application/json"
echo "Cache-Control: no-store, no-cache, must-revalidate"
echo "Pragma: no-cache"

# 防暴力破解設置
BLOCKED_IPS_FILE="/var/tmp/blocked_ips"
FAILED_ATTEMPTS_DIR="/var/tmp/failed_attempts"
MAX_ATTEMPTS=5
BLOCK_TIME=600 # 10分鐘（秒）

# 確保目錄和文件存在
mkdir -p "$FAILED_ATTEMPTS_DIR"
touch "$BLOCKED_IPS_FILE"
chmod 600 "$BLOCKED_IPS_FILE"
chmod 700 "$FAILED_ATTEMPTS_DIR"

# 會話驗證函數
validate_session() {
    local session_id="$1"
    local session_file="$SESSION_DIR/$session_id"
    local current_time=$(date +%s)

    # 檢查會話是否存在且有效
    if [[ -n "$session_id" ]] && [[ -f "$session_file" ]]; then
        # 讀取會話創建時間
        local created_time=$(stat -c %Y "$session_file")
        local session_age=$((current_time - created_time))

        # 檢查會話是否過期（7天）
        if ((session_age < 604800)); then
            # 更新會話文件的訪問時間
            touch "$session_file"
            return 0
        else
            # 刪除過期會話
            rm -f "$session_file"
            return 2
        fi
    fi
    return 1
}

# 解析 Cookie 函數
parse_cookies() {
    if [[ -n "$HTTP_COOKIE" ]]; then
        local cookies
        declare -gA COOKIES
        IFS=';' read -ra cookies <<<"$HTTP_COOKIE"
        for cookie in "${cookies[@]}"; do
            cookie=$(echo "$cookie" | xargs) # 去除空白
            local key="${cookie%%=*}"
            local value="${cookie#*=}"
            COOKIES["$key"]="$value"
        done
    fi
}

# 檢查 IP 是否被封禁
check_ip_blocked() {
    local ip="$1"
    local current_time=$(date +%s)
    local block_info

    # 清理過期的封禁記錄
    local tmp_file=$(mktemp)
    while IFS=':' read -r blocked_ip block_time; do
        if [[ -n "$blocked_ip" ]] && ((current_time - block_time < BLOCK_TIME)); then
            echo "$blocked_ip:$block_time" >>"$tmp_file"
        fi
    done <"$BLOCKED_IPS_FILE"
    mv "$tmp_file" "$BLOCKED_IPS_FILE"

    # 檢查當前 IP 是否被封禁
    block_info=$(grep "^$ip:" "$BLOCKED_IPS_FILE")
    if [[ -n "$block_info" ]]; then
        block_time=$(echo "$block_info" | cut -d':' -f2)
        remaining_time=$((BLOCK_TIME - (current_time - block_time)))
        if ((remaining_time > 0)); then
            return "$remaining_time"
        fi
    fi
    return 0
}

# 記錄失敗嘗試
record_failed_attempt() {
    local ip="$1"
    local ip_file="$FAILED_ATTEMPTS_DIR/$ip"
    local current_time=$(date +%s)

    # 創建或更新失敗記錄
    echo "$current_time" >>"$ip_file"

    # 只保留最近的失敗記錄
    local attempts=$(tail -n "$MAX_ATTEMPTS" "$ip_file" | wc -l)
    if ((attempts >= MAX_ATTEMPTS)); then
        local oldest_time=$(head -n 1 "$ip_file")
        if ((current_time - oldest_time < BLOCK_TIME)); then
            # 封禁 IP
            echo "$ip:$current_time" >>"$BLOCKED_IPS_FILE"
            rm -f "$ip_file" # 清除失敗記錄
            return 1
        fi
    fi
    return 0
}

# 清除失敗記錄
clear_failed_attempts() {
    local ip="$1"
    rm -f "$FAILED_ATTEMPTS_DIR/$ip"
}

# 日誌函數
log_security() {
    local action="$1"
    local status="$2"
    local message="$3"
    local log_file="/var/log/security.log"
    local log_dir="/var/log"

    # 確保日誌目錄存在且權限正確
    [ ! -d "$log_dir" ] && mkdir -p "$log_dir"
    [ ! -f "$log_file" ] && touch "$log_file"
    chmod 640 "$log_file"

    # 獲取當前月份
    local current_month=$(date +%Y%m)
    local log_month=""
    [ -f "$log_file" ] && log_month=$(date -r "$log_file" +%Y%m)

    # 如果是新的月份，進行日誌輪替
    if [ "$current_month" != "$log_month" ] && [ -f "$log_file" ]; then
        mv "$log_file" "${log_file}.${log_month}"
        gzip "${log_file}.${log_month}"
        touch "$log_file"
        chmod 640 "$log_file"
    fi

    # 格式化日誌條目
    printf "[%s] %s - %s - \"%s %s\" - \"%s\" - %s - %s\n" \
        "$(date '+%Y-%m-%d %H:%M:%S')" \
        "${REMOTE_ADDR:-unknown}" \
        "${action}" \
        "${REQUEST_METHOD:-GET}" \
        "${REQUEST_URI:-unknown}" \
        "${HTTP_USER_AGENT:-unknown}" \
        "${status}" \
        "${message}" \
        >>"$log_file"
}

# 設置安全的 Cookie
set_secure_cookie() {
    local name="$1"
    local value="$2"
    local max_age="$3"
    local domain="${4:-}"
    local same_site="${5:-Strict}"

    local cookie="Set-Cookie: ${name}=${value}; Path=/; HttpOnly; Secure"

    # 添加可選參數
    [[ -n "$max_age" ]] && cookie+="; Max-Age=${max_age}"
    [[ -n "$domain" ]] && cookie+="; Domain=${domain}"
    cookie+="; SameSite=${same_site}"

    echo "$cookie"
}

# 安全檢查函數
check_security() {
    # 檢查是否使用 HTTPS
    if [ "$HTTPS" != "on" ]; then
        log_security "$action" "error" "非 HTTPS 連接"
        echo "{"
        echo "  \"status\": \"error\","
        echo "  \"message\": \"必須使用 HTTPS 連接\""
        echo "}"
        exit 1
    fi

    # 檢查 HTTP 方法
    if [ "$REQUEST_METHOD" != "GET" ] && [ "$REQUEST_METHOD" != "POST" ]; then
        log_security "$action" "error" "不支援的請求方法: ${REQUEST_METHOD}"
        echo "{"
        echo "  \"status\": \"error\","
        echo "  \"message\": \"不支援的請求方法\""
        echo "}"
        exit 1
    fi

    # 檢查 IP 是否被封禁
    local remaining_time
    check_ip_blocked "$REMOTE_ADDR"
    remaining_time=$?
    if ((remaining_time > 0)); then
        log_security "$action" "error" "IP 已被封禁，剩餘時間: ${remaining_time} 秒"
        echo "{"
        echo "  \"status\": \"error\","
        echo "  \"message\": \"由於多次失敗嘗試，您的 IP 已被暫時封禁\","
        echo "  \"remaining_time\": $remaining_time"
        echo "}"
        exit 1
    fi
}

# 驗證會話並返回結果
check_session_status() {
    local session_id="$1"
    local result

    validate_session "$session_id"
    result=$?

    case $result in
    0) # 會話有效
        log_security "validate" "success" "會話有效: ${session_id}"
        echo "{"
        echo "  \"status\": \"valid\","
        echo "  \"message\": \"Session is valid.\""
        echo "}"
        ;;
    2) # 會話過期
        log_security "validate" "error" "會話已過期: ${session_id}"
        # 清除過期的 Cookie
        set_secure_cookie "SESSION_ID" "" "0"
        echo
        echo "{"
        echo "  \"status\": \"invalid\","
        echo "  \"message\": \"Session has expired.\""
        echo "}"
        ;;
    *) # 會話無效
        log_security "validate" "error" "會話無效或不存在"
        echo "{"
        echo "  \"status\": \"invalid\","
        echo "  \"message\": \"Not logged in.\""
        echo "}"
        ;;
    esac
}

# 解析 QUERY_STRING
IFS='&' read -r -a params <<<"$QUERY_STRING"
declare -A query
for param in "${params[@]}"; do
    IFS='=' read -r key value <<<"$param"
    query[$key]=$(echo "$value" | sed 's/+/ /g' | sed 's/%\([0-9A-F][0-9A-F]\)/\\x\1/g' | xargs -0 echo -e)
done

action="${query[action]}"
redirect="${query[redirect]:-/}"

# 會話文件存儲目錄
SESSION_DIR="/tmp/sessions"
RESET_DIR="/tmp/resets"
mkdir -p "$SESSION_DIR" "$RESET_DIR"
chmod 700 "$SESSION_DIR" "$RESET_DIR"

# 執行安全檢查
check_security

# 解析 Cookie
parse_cookies

case "$action" in
"login")
    # 檢查登入憑證
    username="${query[username]}"
    password="${query[password]}"

    # 模擬登入驗證（實際應用中應該進行真實的驗證）
    if [[ "$username" == "admin" ]] && [[ "$password" == "password" ]]; then
        # 登入成功，清除失敗記錄
        clear_failed_attempts "$REMOTE_ADDR"

        # 生成新會話ID
        new_session_id=$(head -c 16 /dev/urandom | xxd -p)
        touch "$SESSION_DIR/$new_session_id"

        # 設置安全的 Cookie（7天過期）
        set_secure_cookie "SESSION_ID" "$new_session_id" "604800"
        echo

        if [[ -n "$redirect" ]]; then
            log_security "login" "success" "登入成功，重定向到 ${redirect}"
            echo "{"
            echo "  \"status\": \"success\","
            echo "  \"message\": \"登入成功\","
            echo "  \"redirect\": \"$redirect\""
            echo "}"
        else
            log_security "login" "success" "登入成功"
            echo "{"
            echo "  \"status\": \"success\","
            echo "  \"message\": \"登入成功\""
            echo "}"
        fi
    else
        # 記錄失敗嘗試
        if record_failed_attempt "$REMOTE_ADDR"; then
            log_security "login" "error" "登入失敗"
            echo "{"
            echo "  \"status\": \"error\","
            echo "  \"message\": \"用戶名或密碼錯誤\""
            echo "}"
        else
            log_security "login" "error" "登入失敗次數過多，IP 已被封禁"
            echo "{"
            echo "  \"status\": \"error\","
            echo "  \"message\": \"由於多次失敗嘗試，您的 IP 已被暫時封禁\","
            echo "  \"block_time\": $BLOCK_TIME"
            echo "}"
        fi
    fi
    ;;

"logout")
    # 刪除會話文件
    if [[ -n "${COOKIES[SESSION_ID]}" ]]; then
        rm -f "$SESSION_DIR/${COOKIES[SESSION_ID]}"
    fi

    # 清除 Cookie
    set_secure_cookie "SESSION_ID" "" "0"
    echo

    log_security "logout" "success" "登出成功"
    echo "{"
    echo "  \"status\": \"success\","
    echo "  \"message\": \"登出成功\""
    echo "}"
    ;;

"check")
    echo
    if validate_session "${COOKIES[SESSION_ID]}"; then
        log_security "check" "success" "會話有效: ${COOKIES[SESSION_ID]}"
        echo "{"
        echo "  \"status\": \"success\","
        echo "  \"logged_in\": true,"
        echo "  \"session_id\": \"${COOKIES[SESSION_ID]}\""
        echo "}"
    else
        log_security "check" "success" "會話無效或不存在"
        echo "{"
        echo "  \"status\": \"success\","
        echo "  \"logged_in\": false"
        echo "}"
    fi
    ;;

"validate")
    echo
    check_session_status "${COOKIES[SESSION_ID]}"
    ;;

*)
    echo
    log_security "$action" "error" "無效的操作"
    echo "{"
    echo "  \"status\": \"error\","
    echo "  \"message\": \"無效的操作\""
    echo "}"
    ;;
esac
