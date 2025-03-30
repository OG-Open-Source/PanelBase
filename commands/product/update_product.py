#!/usr/bin/env python3
import json
import argparse
import sys
from datetime import datetime

# 解析命令行参数
parser = argparse.ArgumentParser(description='更新产品信息')
parser.add_argument('--product_id', required=True, help='产品ID')
parser.add_argument('--name', required=True, help='产品名称')
parser.add_argument('--price', required=True, help='产品价格')
args = parser.parse_args()

# 模拟数据库操作
def update_product(product_id, name, price):
	# 这里只是示例，实际应用中应更新数据库
	products = {
		"456": {
			"id": "456",
			"name": "旧产品名称",
			"price": 19.99,
			"description": "产品描述",
			"category": "电子产品",
			"inventory": 100,
			"created_at": "2023-05-10T14:30:00Z",
			"updated_at": "2023-06-15T09:20:00Z"
		}
	}
	
	if product_id not in products:
		return None
	
	# 更新产品信息
	product = products[product_id]
	product["name"] = name
	try:
		product["price"] = float(price)
	except ValueError:
		return {"error": "Price must be a valid number"}
	
	product["updated_at"] = datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ")
	
	return product

# 更新产品并返回结果
result = update_product(args.product_id, args.name, args.price)
if result:
	if "error" in result:
		response = {
			"success": False,
			"error": result["error"]
		}
	else:
		response = {
			"success": True,
			"product": result,
			"message": "Product updated successfully"
		}
else:
	response = {
		"success": False,
		"error": f"Product not found with ID: {args.product_id}"
	}

# 输出JSON格式的结果
print(json.dumps(response, ensure_ascii=False)) 