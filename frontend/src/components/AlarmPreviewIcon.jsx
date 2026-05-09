import { Badge, Drawer, List, Typography } from 'antd';
import { BellOutlined } from '@ant-design/icons';
import { useState } from 'react';

const { Text } = Typography;

export default function AlarmPreviewIcon() {
  const [open, setOpen] = useState(false);
  // Mock: chưa có alarm thực
  const alarmCount = 0;
  const alarms = [];

  return (
    <>
      <Badge count={alarmCount} size="small" offset={[-2, 2]}>
        <BellOutlined
          style={{ fontSize: 18, cursor: 'pointer' }}
          onClick={() => setOpen(true)}
        />
      </Badge>
      <Drawer
        title="Cảnh báo"
        placement="right"
        onClose={() => setOpen(false)}
        open={open}
        width={360}
      >
        {alarms.length === 0 ? (
          <Text type="secondary">Không có cảnh báo nào</Text>
        ) : (
          <List
            dataSource={alarms}
            renderItem={(item) => (
              <List.Item>
                <Text>{item.message}</Text>
              </List.Item>
            )}
          />
        )}
      </Drawer>
    </>
  );
}

/// Biểu tượng chuông cảnh báo với badge đếm số lượng. 
// Hiện tại là mock dữ liệu (số 0). Sau này khi có WebSocket “alarm“ sẽ cập nhật. 
// Khi click vào sẽ mở Drawer hiển thị danh sách cảnh báo chi tiết. 
// Dùng trong Header, bên cạnh LatencyBadge.

// Tương lai: Khi có WebSocket gửi alarm, component này sẽ subscribe và cập nhật alarmCount và danh sách alarms.