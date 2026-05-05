#!/bin/bash
set -e

EMQX_HOST="http://localhost:18083"
ADMIN_AUTH="admin:Admin@123"

# Đợi EMQX sẵn sàng
echo "⏳ Chờ EMQX API..."
until curl -s -f -o /dev/null "${EMQX_HOST}/api/v5/status"; do sleep 2; done
echo "✅ EMQX sẵn sàng."

# ------------------------------------------
# Hàm gọi API
# ------------------------------------------
api() {
    curl -s -u "$ADMIN_AUTH" "$@"
}

# ------------------------------------------
# 1. Tạo người dùng
# ------------------------------------------
echo "👤 Tạo user core-service..."
api -X POST "${EMQX_HOST}/api/v5/authentication/password_based:built_in_database/users" \
    -H "Content-Type: application/json" \
    -d '{"user_id":"core-service","password":"CoreService@2025"}' || echo "   (có thể đã tồn tại)"

echo "👤 Tạo user gateway-sim..."
api -X POST "${EMQX_HOST}/api/v5/authentication/password_based:built_in_database/users" \
    -H "Content-Type: application/json" \
    -d '{"user_id":"gateway-sim","password":"GatewaySim@2025"}' || echo "   (có thể đã tồn tại)"

# ------------------------------------------
# 2. Xóa tất cả rule cũ trong built_in_database
# ------------------------------------------
echo "🧹 Xóa rule cũ..."
RULES=$(api "${EMQX_HOST}/api/v5/authorization/sources/built_in_database/rules" | jq -r '.data[]?.id')
if [ -n "$RULES" ]; then
    for id in $RULES; do
        api -X DELETE "${EMQX_HOST}/api/v5/authorization/sources/built_in_database/rules/${id}"
    done
    echo "   Đã xóa."
else
    echo "   Không có rule cũ."
fi

# ------------------------------------------
# 3. Thêm rule mới cho core-service
# ------------------------------------------
echo "🔐 Thêm rule cho core-service..."

# Subscribe tenant/+/device/+/data và heartbeat
api -X POST "${EMQX_HOST}/api/v5/authorization/sources/built_in_database/rules" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "allow",
        "username": "core-service",
        "action": "subscribe",
        "topics": ["tenant/+/device/+/data", "tenant/+/device/+/heartbeat"]
    }'

# Publish data/# và metadata/#
api -X POST "${EMQX_HOST}/api/v5/authorization/sources/built_in_database/rules" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "allow",
        "username": "core-service",
        "action": "publish",
        "topics": ["data/#", "metadata/#"]
    }'

# ------------------------------------------
# 4. Thêm rule cho gateway-sim
# ------------------------------------------
echo "🔐 Thêm rule cho gateway-sim..."

# Publish tenant/+/device/+/data và heartbeat
api -X POST "${EMQX_HOST}/api/v5/authorization/sources/built_in_database/rules" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "allow",
        "username": "gateway-sim",
        "action": "publish",
        "topics": ["tenant/+/device/+/data", "tenant/+/device/+/heartbeat"]
    }'

# Subscribe tenant/+/device/+/cmd (tương lai)
api -X POST "${EMQX_HOST}/api/v5/authorization/sources/built_in_database/rules" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "allow",
        "username": "gateway-sim",
        "action": "subscribe",
        "topics": ["tenant/+/device/+/cmd"]
    }'

echo "🎉 Hoàn tất! Rule đã được thêm vào built_in_database."