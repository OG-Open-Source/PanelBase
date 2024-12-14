#!/usr/bin/env python3
import json
import os
import sys
import platform

# 設置 HTTP 頭
print("Content-Type: application/json")
print()

# 獲取系統信息
system_info = {
    "os": platform.system(),
    "platform": platform.platform(),
    "python_version": sys.version,
    "hostname": platform.node(),
    "cpu_arch": platform.machine(),
    "environment": dict(os.environ)
}

# 輸出 JSON 格式的響應
print(json.dumps(system_info, indent=2, ensure_ascii=False)) 