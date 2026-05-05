# nlscada-cloud
NL SCADA Cloud là trung tâm giám sát và thu thập dữ liệu thời gian thực dành cho hệ sinh thái NL SCADA. Dự án cung cấp một nền tảng nhẹ, dễ triển khai, cho phép kết nối các thiết bị gateway, lưu trữ dữ liệu chuỗi thời gian, quản lý metadata và hiển thị trực quan trên giao diện web.
# NL SCADA Cloud

Trung tâm giám sát và thu thập dữ liệu thời gian thực cho hệ sinh thái NL SCADA.

## Kiến trúc

- **NL SCADA Core**: Go monolith (Ingest, API, WebSocket)
- **WebBase**: React SPA
- **EMQX**: MQTT Broker
- **PostgreSQL**: Metadata
- **InfluxDB**: Time-series data

## Bắt đầu nhanh

```bash
docker-compose -f deployments/docker-compose.yaml up -d
Xem chi tiết trong docs/.
```