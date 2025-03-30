#!/bin/bash
##############################
# PanelBase Token 管理工具腳本 (Linux版)
# 用於獲取 JWT 令牌和創建 API 令牌
##############################

# 全局變量
BASE_URL="http://localhost:45784"  # 默認 URL，可修改為您的實際 URL
JWT_TOKEN=""
API_TOKEN=""
USERNAME=""
PASSWORD=""
TOKEN_EXPIRATION=""

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

show_header() {
    echo -e "${CYAN}====================================${NC}"
    echo -e "${CYAN}    PanelBase Token 管理工具 (Linux版)${NC}"
    echo -e "${CYAN}====================================${NC}"
    echo
}

show_menu() {
    echo -e "${YELLOW}請選擇操作:${NC}"
    echo -e "${GREEN}1. 設置服務器 URL${NC}"
    echo -e "${GREEN}2. 登錄並獲取 JWT 令牌${NC}"
    echo -e "${GREEN}3. 創建新的 API 令牌${NC}"
    echo -e "${GREEN}4. 列出當前令牌${NC}"
    echo -e "${GREEN}5. 測試令牌有效性${NC}"
    echo -e "${GREEN}6. 保存令牌到文件${NC}"
    echo -e "${GREEN}0. 退出${NC}"
    echo
}

set_server_url() {
    echo -e "${MAGENTA}當前服務器 URL: ${BASE_URL}${NC}"
    echo -n "請輸入新的服務器 URL (直接回車保持當前值): "
    read new_url
    
    if [ ! -z "$new_url" ]; then
        BASE_URL="$new_url"
        echo -e "${GREEN}服務器 URL 已更新為: ${BASE_URL}${NC}"
    fi
}

get_jwt_token() {
    echo -e "${MAGENTA}登錄並獲取 JWT 令牌${NC}"
    
    echo -n "用戶名: "
    read USERNAME
    echo -n "密碼: "
    read -s PASSWORD
    echo
    
    # 獲取過期時間設置
    echo -n "令牌過期時間 (小時, 直接回車使用默認值 24 小時): "
    read TOKEN_EXPIRATION
    if [ -z "$TOKEN_EXPIRATION" ]; then
        TOKEN_EXPIRATION="24"
    fi
    
    login_url="${BASE_URL}/api/v1/auth/login"
    
    # 使用curl獲取JWT令牌
    response=$(curl -s -X POST $login_url \
      -H "Content-Type: application/json" \
      -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\",\"duration\":\"${TOKEN_EXPIRATION}\"}")
    
    # 解析響應（使用grep和cut簡單處理，生產環境建議使用jq）
    status=$(echo $response | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    
    if [ "$status" = "success" ]; then
        JWT_TOKEN=$(echo $response | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        user_name=$(echo $response | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
        user_username=$(echo $response | grep -o '"username":"[^"]*"' | cut -d'"' -f4)
        user_role=$(echo $response | grep -o '"role":"[^"]*"' | cut -d'"' -f4)
        expires=$(echo $response | grep -o '"expires":"[^"]*"' | cut -d'"' -f4)
        
        echo -e "${GREEN}登錄成功! JWT 令牌已獲取。${NC}"
        echo -e "${GREEN}用戶: ${user_name} (${user_username})${NC}"
        echo -e "${GREEN}角色: ${user_role}${NC}"
        echo -e "${GREEN}過期時間: ${expires} 小時${NC}"
        echo
        echo -e "${CYAN}JWT 令牌:${NC}"
        echo -e "${GRAY}${JWT_TOKEN}${NC}"
        
        return 0
    else
        message=$(echo $response | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
        echo -e "${RED}登錄失敗: ${message}${NC}"
        return 1
    fi
}

create_api_token() {
    if [ -z "$JWT_TOKEN" ]; then
        echo -e "${RED}請先登錄獲取 JWT 令牌${NC}"
        return 1
    fi
    
    echo -e "${MAGENTA}創建新的 API 令牌${NC}"
    
    echo -n "令牌名稱: "
    read token_name
    echo -n "權限 (多個權限用逗號分隔, 如: read,write): "
    read token_permissions
    
    echo -n "有效期 (ISO 8601格式, 如: PT1H 表示1小時, PT7D 表示7天): "
    read token_duration
    if [ -z "$token_duration" ]; then
        token_duration="PT1H"  # 默認1小時
    fi
    
    echo -n "速率限制 (每分鐘請求數, 默認 60): "
    read token_rate_limit
    if [ -z "$token_rate_limit" ]; then
        token_rate_limit=60
    fi
    
    # 處理權限數組
    IFS=',' read -ra permissions_array <<< "$token_permissions"
    permissions_json=""
    for perm in "${permissions_array[@]}"; do
        if [ -z "$permissions_json" ]; then
            permissions_json="\"$(echo $perm | xargs)\""
        else
            permissions_json="${permissions_json},\"$(echo $perm | xargs)\""
        fi
    done
    
    token_url="${BASE_URL}/api/v1/auth/token"
    
    response=$(curl -s -X POST $token_url \
      -H "Authorization: Bearer ${JWT_TOKEN}" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"${token_name}\",\"permissions\":[${permissions_json}],\"duration\":\"${token_duration}\",\"rate_limit\":${token_rate_limit}}")
    
    status=$(echo $response | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    
    if [ "$status" = "success" ]; then
        API_TOKEN=$(echo $response | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        created_name=$(echo $response | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
        
        echo -e "${GREEN}API 令牌創建成功!${NC}"
        echo -e "${GREEN}令牌名稱: ${created_name}${NC}"
        echo
        echo -e "${CYAN}API 令牌:${NC}"
        echo -e "${GRAY}${API_TOKEN}${NC}"
    else
        message=$(echo $response | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
        echo -e "${RED}創建 API 令牌失敗: ${message}${NC}"
        
        # 檢查是否是令牌過期錯誤
        if [[ $response == *"token is expired"* ]]; then
            echo -e "${YELLOW}JWT令牌可能已過期，嘗試重新登錄...${NC}"
            if get_jwt_token; then
                echo -e "${YELLOW}重新嘗試創建 API 令牌...${NC}"
                create_api_token
            fi
        fi
    fi
}

show_current_tokens() {
    echo -e "${MAGENTA}當前令牌信息:${NC}"
    
    if [ ! -z "$JWT_TOKEN" ]; then
        echo -e "${GREEN}JWT 令牌:${NC}"
        echo -e "${GRAY}${JWT_TOKEN}${NC}"
    else
        echo -e "${YELLOW}未獲取 JWT 令牌${NC}"
    fi
    
    echo
    
    if [ ! -z "$API_TOKEN" ]; then
        echo -e "${GREEN}API 令牌:${NC}"
        echo -e "${GRAY}${API_TOKEN}${NC}"
    else
        echo -e "${YELLOW}未創建 API 令牌${NC}"
    fi
}

test_token_validity() {
    echo -e "${MAGENTA}測試令牌有效性${NC}"
    
    echo -n "要測試的令牌類型 (1: JWT, 2: API): "
    read token_type
    token=""
    
    case $token_type in
        1)
            token="$JWT_TOKEN"
            if [ -z "$token" ]; then
                echo -e "${RED}未獲取 JWT 令牌${NC}"
                return 1
            fi
            ;;
        2)
            token="$API_TOKEN"
            if [ -z "$token" ]; then
                echo -e "${RED}未創建 API 令牌${NC}"
                return 1
            fi
            ;;
        *)
            echo -e "${RED}無效的選擇${NC}"
            return 1
            ;;
    esac
    
    test_url="${BASE_URL}/api/v1/users"
    
    response=$(curl -s -X GET $test_url \
      -H "Authorization: Bearer ${token}")
    
    status=$(echo $response | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    
    if [ "$status" = "success" ]; then
        echo -e "${GREEN}令牌有效! 成功訪問資源。${NC}"
        echo -e "${GREEN}響應狀態: ${status}${NC}"
    else
        message=$(echo $response | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
        echo -e "${RED}令牌無效或已過期${NC}"
        echo -e "${RED}錯誤: ${message}${NC}"
    fi
}

save_tokens() {
    output_path="./tokens.txt"
    
    echo "# PanelBase 令牌信息" > $output_path
    echo "# 生成時間: $(date)" >> $output_path
    echo "# 服務器: ${BASE_URL}" >> $output_path
    echo "" >> $output_path
    
    if [ ! -z "$JWT_TOKEN" ]; then
        echo "# JWT令牌" >> $output_path
        echo "$JWT_TOKEN" >> $output_path
        echo "" >> $output_path
    fi
    
    if [ ! -z "$API_TOKEN" ]; then
        echo "# API令牌" >> $output_path
        echo "$API_TOKEN" >> $output_path
        echo "" >> $output_path
    fi
    
    echo "# 使用示例:" >> $output_path
    echo "# curl -H \"Authorization: Bearer {token}\" ${BASE_URL}/api/v1/users" >> $output_path
    
    echo -e "${GREEN}令牌已保存到文件: ${output_path}${NC}"
}

start_token_manager() {
    clear
    show_header
    
    while true; do
        show_menu
        echo -n "請輸入選項: "
        read choice
        
        case $choice in
            1)
                set_server_url
                ;;
            2)
                get_jwt_token
                ;;
            3)
                create_api_token
                ;;
            4)
                show_current_tokens
                ;;
            5)
                test_token_validity
                ;;
            6)
                save_tokens
                ;;
            0)
                echo -e "${CYAN}感謝使用 PanelBase Token 管理工具${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}無效的選項，請重試${NC}"
                ;;
        esac
        
        echo
        echo -n "按 Enter 鍵繼續..."
        read
        clear
        show_header
    done
}

# 檢查必要命令
command -v curl >/dev/null 2>&1 || { echo -e "${RED}錯誤: 需要安裝 curl 但未找到。請安裝後再試。${NC}"; exit 1; }

# 啟動程序
start_token_manager 