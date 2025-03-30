#!/usr/bin/env python3
import json
import argparse
import sys

# 解析命令行参数
parser = argparse.ArgumentParser(description='获取用户信息')
parser.add_argument('--user_id', required=True, help='用户ID')
args = parser.parse_args()

# 模拟数据库查询
def get_user(user_id):
	# 这里只是示例，实际应用中应查询数据库
	users = {
		"123": {
			"id": "123",
			"username": "johndoe",
			"email": "john.doe@example.com",
			"created_at": "2023-01-15T08:00:00Z",
			"status": "active",
			"role": "admin"
		},
		"456": {
			"id": "456",
			"username": "janedoe",
			"email": "jane.doe@example.com",
			"created_at": "2023-02-20T10:30:00Z",
			"status": "active",
			"role": "user"
		}
	}
	
	if user_id in users:
		return users[user_id]
	else:
		return None

# 获取用户并返回结果
user = get_user(args.user_id)
if user:
	response = {
		"success": True,
		"user": user
	}
else:
	response = {
		"success": False,
		"error": f"User not found with ID: {args.user_id}"
	}

# 输出JSON格式的结果
print(json.dumps(response, ensure_ascii=False)) 