#!/usr/bin/env python
# -*- coding: utf-8 -*-

import json
import sys
import datetime

# 简单的数据生成脚本
# 这个脚本可以通过路由脚本系统被调用

def get_data():
	"""返回一些示例数据"""
	data = {
		"timestamp": datetime.datetime.now().isoformat(),
		"generator": "PanelBase Route Script",
		"data": {
			"numbers": [1, 2, 3, 4, 5],
			"strings": ["hello", "world", "from", "script"],
			"nested": {
				"boolean": True,
				"null": None
			}
		}
	}

	# 检查是否有参数
	if len(sys.argv) > 1:
		data["args"] = sys.argv[1:]

	return data

if __name__ == "__main__":
	# 输出JSON格式的数据
	print(json.dumps(get_data(), indent=2))