import { Tag } from 'antd';
import { ClockCircleOutlined } from '@ant-design/icons';

// Tạm thời sử dụng một giá trị static cho đến khi có hook thực sự
// Khi có useConnectionStatus, import và dùng: const { latency } = useConnectionStatus();

export default function LatencyBadge() {
  // Mock latency, sẽ thay bằng hook sau
  const latency = 0; // useConnectionStatus().latency ?? 0;

  let color = 'green';
  if (latency > 2000) color = 'red';
  else if (latency > 500) color = 'orange';

  return (
    <Tag icon={<ClockCircleOutlined />} color={color}>
      {latency > 0 ? `${latency}ms` : '---'}
    </Tag>
  );
}

//Hiển thị độ trễ kết nối dữ liệu (tính bằng mili giây). Sử dụng useConnectionStatus hook (sẽ tạo sau). Nếu hook chưa sẵn sàng, component sẽ không crash mà hiển thị giá trị mặc định.