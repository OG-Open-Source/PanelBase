#!/bin/bash
##############################
# PanelBase Token 解析工具 (Bash版)
# 用於解析 JWT 令牌和 API 令牌的結構
##############################

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
GRAY='\033[0;37m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

show_banner() {
    echo -e "${CYAN}====================================${NC}"
    echo -e "${CYAN}    PanelBase Token 解析工具 (Bash版) ${NC}"
    echo -e "${CYAN}====================================${NC}"
    echo
}

# Base64URL解碼函數
decode_base64url() {
    local input=$1
    
    # 替換URL安全字符為標準Base64字符
    input=$(echo $input | tr '-_' '+/')
    
    # 添加填充
    remainder=$((${#input} % 4))
    if [ $remainder -eq 2 ]; then
        input="${input}=="
    elif [ $remainder -eq 3 ]; then
        input="${input}="
    fi
    
    # 解碼Base64並輸出
    echo "$input" | base64 -d 2>/dev/null
}

# 解析Unix時間戳為人類可讀格式
parse_timestamp() {
    local timestamp=$1
    if [[ $timestamp =~ ^[0-9]+$ ]]; then
        if command -v date >/dev/null 2>&1; then
            if date --version >/dev/null 2>&1; then
                # GNU date (Linux)
                date -d @$timestamp "+%Y-%m-%d %H:%M:%S %Z"
            else
                # BSD date (macOS)
                date -r $timestamp "+%Y-%m-%d %H:%M:%S %Z"
            fi
        else
            echo "$timestamp (無法轉換，date命令不可用)"
        fi
    else
        echo "未設置"
    fi
}

# 檢查令牌過期狀態
check_token_expiry() {
    local exp_time=$1
    local now=$(date +%s)
    
    if [[ $exp_time =~ ^[0-9]+$ ]]; then
        if [ $exp_time -lt $now ]; then
            echo -e "${RED}已過期${NC}"
        else
            local remaining=$(( ($exp_time - $now) / 3600 ))
            echo -e "${GREEN}有效 (剩餘約 $remaining 小時)${NC}"
        fi
    else
        echo -e "${YELLOW}無法確定${NC}"
    fi
}

# 彩色輸出JSON
format_json() {
    if command -v jq >/dev/null 2>&1; then
        echo "$1" | jq .
    else
        echo "$1" | python3 -m json.tool 2>/dev/null || python -m json.tool 2>/dev/null || echo "$1"
    fi
}

# 解析JWT令牌
parse_jwt_token() {
    local token=$1
    
    # 分割令牌
    IFS='.' read -r header_base64 payload_base64 signature <<< "$token"
    
    if [ -z "$header_base64" ] || [ -z "$payload_base64" ] || [ -z "$signature" ]; then
        echo -e "${RED}錯誤: 無效的JWT格式。JWT應該包含3部分 (標頭.載荷.簽名)${NC}"
        return 1
    fi
    
    # 解碼標頭
    header_json=$(decode_base64url "$header_base64")
    
    # 解碼載荷
    payload_json=$(decode_base64url "$payload_base64")
    
    # 顯示解析結果
    echo -e "${YELLOW}===== JWT令牌解析結果 =====${NC}"
    
    echo -e "\n${GREEN}[標頭 (Header)]${NC}"
    echo -e "${GRAY}原始數據: $header_base64${NC}"
    echo -e "解碼後:"
    format_json "$header_json"
    
    echo -e "\n${GREEN}[載荷 (Payload)]${NC}"
    echo -e "${GRAY}原始數據: $payload_base64${NC}"
    echo -e "解碼後:"
    format_json "$payload_json"
    
    echo -e "\n${GREEN}[簽名 (Signature)]${NC}"
    echo -e "${GRAY}$signature${NC}"
    
    # 提取關鍵字段
    echo -e "\n${MAGENTA}[關鍵信息]${NC}"
    
    # 使用Python或jq提取字段（如果可用）
    if command -v jq >/dev/null 2>&1; then
        # 使用jq提取
        type=$(echo "$payload_json" | jq -r '.type // "未設置"')
        user_id=$(echo "$payload_json" | jq -r '.user_id // "未設置"')
        username=$(echo "$payload_json" | jq -r '.username // "未設置"')
        role=$(echo "$payload_json" | jq -r '.role // "未設置"')
        subject=$(echo "$payload_json" | jq -r '.sub // "未設置"')
        exp=$(echo "$payload_json" | jq -r '.exp // "未設置"')
        iat=$(echo "$payload_json" | jq -r '.iat // "未設置"')
        jti=$(echo "$payload_json" | jq -r '.jti // "未設置"')
        token_id=$(echo "$payload_json" | jq -r '.token_id // "未設置"')
    elif command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1; then
        # 使用Python提取
        python_cmd=$(command -v python3 || command -v python)
        type=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('type', '未設置'))" <<< "$payload_json")
        user_id=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('user_id', '未設置'))" <<< "$payload_json")
        username=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('username', '未設置'))" <<< "$payload_json")
        role=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('role', '未設置'))" <<< "$payload_json")
        subject=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('sub', '未設置'))" <<< "$payload_json")
        exp=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('exp', '未設置'))" <<< "$payload_json")
        iat=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('iat', '未設置'))" <<< "$payload_json")
        jti=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('jti', '未設置'))" <<< "$payload_json")
        token_id=$($python_cmd -c "import sys,json; data=json.loads(sys.stdin.read()); print(data.get('token_id', '未設置'))" <<< "$payload_json")
    else
        # 使用grep和cut的基本提取（不太可靠但是一個後備選項）
        type=$(echo "$payload_json" | grep -o '"type":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
        user_id=$(echo "$payload_json" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
        username=$(echo "$payload_json" | grep -o '"username":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
        role=$(echo "$payload_json" | grep -o '"role":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
        subject=$(echo "$payload_json" | grep -o '"sub":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
        exp=$(echo "$payload_json" | grep -o '"exp":[0-9]*' | cut -d':' -f2 || echo "未設置")
        iat=$(echo "$payload_json" | grep -o '"iat":[0-9]*' | cut -d':' -f2 || echo "未設置")
        jti=$(echo "$payload_json" | grep -o '"jti":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
        token_id=$(echo "$payload_json" | grep -o '"token_id":"[^"]*"' | cut -d'"' -f4 || echo "未設置")
    fi
    
    # 顯示提取的字段
    if [ "$type" != "未設置" ]; then
        echo -e "${CYAN}令牌類型: $type${NC}"
    fi
    
    if [ "$user_id" != "未設置" ]; then
        echo -e "${CYAN}用戶ID: $user_id${NC}"
    fi
    
    if [ "$subject" != "未設置" ]; then
        echo -e "${CYAN}主題 (Subject): $subject${NC}"
    fi
    
    if [ "$username" != "未設置" ]; then
        echo -e "${CYAN}用戶名: $username${NC}"
    fi
    
    if [ "$role" != "未設置" ]; then
        echo -e "${CYAN}角色: $role${NC}"
    fi
    
    if [ "$exp" != "未設置" ]; then
        exp_date=$(parse_timestamp "$exp")
        echo -e "${CYAN}過期時間: $exp_date${NC}"
        echo -e "${CYAN}狀態: $(check_token_expiry "$exp")${NC}"
    fi
    
    if [ "$iat" != "未設置" ]; then
        iat_date=$(parse_timestamp "$iat")
        echo -e "${CYAN}簽發時間: $iat_date${NC}"
    fi
    
    if [ "$jti" != "未設置" ]; then
        echo -e "${CYAN}JWT ID: $jti${NC}"
    fi
    
    if [ "$token_id" != "未設置" ]; then
        echo -e "${CYAN}令牌ID: $token_id${NC}"
    fi
    
    return 0
}

# 保存解析結果到文件
save_parsed_token() {
    local token=$1
    local output_file=$2
    
    if [ -z "$output_file" ]; then
        output_file="token_details.json"
    fi
    
    # 分割令牌
    IFS='.' read -r header_base64 payload_base64 signature <<< "$token"
    
    # 解碼標頭和載荷
    header_json=$(decode_base64url "$header_base64")
    payload_json=$(decode_base64url "$payload_base64")
    
    # 創建輸出JSON
    if command -v jq >/dev/null 2>&1; then
        echo "{\"token\":\"$token\",\"header\":$header_json,\"payload\":$payload_json,\"signature\":\"$signature\"}" | jq . > "$output_file"
    elif command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1; then
        python_cmd=$(command -v python3 || command -v python)
        $python_cmd -c "
import json, sys
data = {
    'token': '$token',
    'header': json.loads('$header_json'),
    'payload': json.loads('$payload_json'),
    'signature': '$signature'
}
print(json.dumps(data, indent=2))
" > "$output_file"
    else
        # 基本輸出（非格式化）
        echo "{\"token\":\"$token\",\"header\":$header_json,\"payload\":$payload_json,\"signature\":\"$signature\"}" > "$output_file"
    fi
    
    echo -e "${GREEN}令牌詳細信息已保存到: $output_file${NC}"
}

# 檢查依賴
check_dependencies() {
    local missing_deps=0
    
    # 檢查base64命令
    if ! command -v base64 >/dev/null 2>&1; then
        echo -e "${RED}錯誤: 未找到'base64'命令，這是解碼令牌所必需的${NC}"
        missing_deps=1
    fi
    
    # 推薦安裝jq，但不是必須的
    if ! command -v jq >/dev/null 2>&1; then
        echo -e "${YELLOW}警告: 未找到'jq'命令。建議安裝jq以獲得更好的JSON解析功能${NC}"
        
        # 檢查是否有Python作為後備
        if ! command -v python3 >/dev/null 2>&1 && ! command -v python >/dev/null 2>&1; then
            echo -e "${YELLOW}警告: 也未找到'python'。JSON解析功能將受限${NC}"
        fi
    fi
    
    return $missing_deps
}

# 主函數
main() {
    show_banner
    
    # 檢查依賴
    check_dependencies
    if [ $? -ne 0 ]; then
        echo -e "${RED}請安裝缺少的依賴後再運行此腳本${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}請選擇輸入方式:${NC}"
    echo -e "${GREEN}1. 直接輸入令牌${NC}"
    echo -e "${GREEN}2. 從文件讀取令牌${NC}"
    echo -e "${GREEN}3. 退出${NC}"
    
    read -p "請選擇: " choice
    
    case $choice in
        1)
            echo -n "請輸入JWT令牌或API令牌: "
            read token
            
            if [ -z "$token" ]; then
                echo -e "${RED}錯誤: 未提供令牌${NC}"
                exit 1
            fi
            
            parse_jwt_token "$token"
            
            if [ $? -eq 0 ]; then
                echo -n "是否保存解析結果到文件? (y/n): "
                read save_choice
                
                if [ "$save_choice" = "y" ]; then
                    echo -n "輸入文件名 (默認: token_details.json): "
                    read file_name
                    
                    if [ -z "$file_name" ]; then
                        file_name="token_details.json"
                    fi
                    
                    save_parsed_token "$token" "$file_name"
                fi
            fi
            ;;
        2)
            echo -n "請輸入令牌文件路徑: "
            read file_path
            
            if [ ! -f "$file_path" ]; then
                echo -e "${RED}錯誤: 文件不存在${NC}"
                exit 1
            fi
            
            # 讀取文件並過濾出令牌（移除註釋和空行）
            tokens=()
            while IFS= read -r line; do
                # 跳過空行和註釋
                if [ -n "$line" ] && ! [[ "$line" =~ ^[[:space:]]*# ]]; then
                    tokens+=("$line")
                fi
            done < "$file_path"
            
            if [ ${#tokens[@]} -eq 0 ]; then
                echo -e "${RED}錯誤: 文件中未找到有效令牌${NC}"
                exit 1
            elif [ ${#tokens[@]} -eq 1 ]; then
                token="${tokens[0]}"
                parse_jwt_token "$token"
                
                if [ $? -eq 0 ]; then
                    echo -n "是否保存解析結果到文件? (y/n): "
                    read save_choice
                    
                    if [ "$save_choice" = "y" ]; then
                        echo -n "輸入文件名 (默認: token_details.json): "
                        read file_name
                        
                        if [ -z "$file_name" ]; then
                            file_name="token_details.json"
                        fi
                        
                        save_parsed_token "$token" "$file_name"
                    fi
                fi
            else
                echo -e "${YELLOW}文件包含多行，請選擇要解析的令牌:${NC}"
                
                for i in "${!tokens[@]}"; do
                    token_preview="${tokens[$i]:0:30}..."
                    echo -e "${GREEN}$((i+1)). $token_preview${NC}"
                done
                
                echo -n "請選擇 (1-${#tokens[@]}): "
                read line_choice
                
                # 驗證輸入
                if ! [[ "$line_choice" =~ ^[0-9]+$ ]] || [ "$line_choice" -lt 1 ] || [ "$line_choice" -gt ${#tokens[@]} ]; then
                    echo -e "${RED}無效的選擇${NC}"
                    exit 1
                fi
                
                token="${tokens[$((line_choice-1))]}"
                parse_jwt_token "$token"
                
                if [ $? -eq 0 ]; then
                    echo -n "是否保存解析結果到文件? (y/n): "
                    read save_choice
                    
                    if [ "$save_choice" = "y" ]; then
                        echo -n "輸入文件名 (默認: token_details.json): "
                        read file_name
                        
                        if [ -z "$file_name" ]; then
                            file_name="token_details.json"
                        fi
                        
                        save_parsed_token "$token" "$file_name"
                    fi
                fi
            fi
            ;;
        3)
            echo -e "${CYAN}退出程序${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}無效選擇，請重試${NC}"
            exit 1
            ;;
    esac
    
    echo -e "\n${CYAN}感謝使用 PanelBase Token 解析工具!${NC}"
}

# 執行主程序
main 