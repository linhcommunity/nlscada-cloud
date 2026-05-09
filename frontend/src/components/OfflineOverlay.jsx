import { Alert } from 'antd';
import { WifiOutlined } from '@ant-design/icons';

export default function OfflineOverlay() {
  return (
    <div
      style={{
        position: 'absolute',
        inset: 0,
        backgroundColor: 'rgba(255, 255, 255, 0.7)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 10,
        backdropFilter: 'blur(2px)',
      }}
    >
      <Alert
        message="Dữ liệu đang tạm dừng"
        description="Mất kết nối tới máy chủ. Đang thử kết nối lại..."
        type="warning"
        icon={<WifiOutlined />}
        showIcon
        style={{ maxWidth: 400 }}
      />
    </div>
  );
}
// Component hiển thị overlay cảnh báo khi mất kết nối dữ liệu thời gian thực. Sử dụng Alert của antd với icon cảnh báo. Overlay có nền mờ và làm mờ nền phía sau để nổi bật thông báo.
// Lớp phủ mờ toàn bộ vùng nội dung khi mất kết nối. Dùng trong RealtimePanel, Layout, hoặc có thể bọc quanh Outlet.