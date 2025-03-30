#!/usr/bin/env python3
import json
import argparse
import sys
from datetime import datetime

# 解析命令行参数
parser = argparse.ArgumentParser(description='删除评论')
parser.add_argument('--comment_id', required=True, help='评论ID')
args = parser.parse_args()

# 模拟数据库操作
def delete_comment(comment_id):
	# 这里只是示例，实际应用中应更新数据库
	comments = {
		"789": {
			"id": "789",
			"user_id": "123",
			"content": "这是一条测试评论",
			"post_id": "555",
			"created_at": "2023-07-05T18:30:00Z"
		}
	}

	if comment_id not in comments:
		return False

	# 删除评论（实际应用中会从数据库删除）
	del comments[comment_id]
	return True

# 删除评论并返回结果
success = delete_comment(args.comment_id)
if success:
	response = {
		"success": True,
		"message": f"Comment {args.comment_id} has been deleted",
		"deleted_at": datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ")
	}
else:
	response = {
		"success": False,
		"error": f"Comment not found with ID: {args.comment_id}"
	}

# 输出JSON格式的结果
print(json.dumps(response, ensure_ascii=False))