```bash
# 登入獲取 JWT token

curl -X POST http://localhost:22450/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}'

# 使用 JWT token 創建 API token

curl -X POST http://localhost:22450/api/v1/auth/token -H "Authorization: Bearer YOUR_JWT_TOKEN" -H "Content-Type: application/json" -d '{"name":"test-token","permissions":["read"],"duration":"PT1H","rate_limit":60}'

# 使用 API token 訪問 API

curl -X GET http://localhost:22450/api/v1/plugins -H "Authorization: Bearer YOUR_API_TOKEN"

# 嘗試未認證訪問

curl -X GET http://localhost:22450/api/v1/plugins

# 預期返回 401 未授權錯誤

# 使用受限權限的 API token 訪問需要其他權限的 API

curl -X POST http://localhost:22450/api/v1/plugins -H "Authorization: Bearer READ_ONLY_API_TOKEN" -H "Content-Type: application/json" -d '{...}'

# 預期返回 403 權限不足錯誤
```