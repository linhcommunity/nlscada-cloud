import { Card, Tag, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { WifiOutlined, ClockCircleOutlined, TagOutlined } from '@ant-design/icons';
import StatusBadge from '../components/StatusBadge';
import { formatRelativeTime } from '../utils/formatters';
import PropTypes from 'prop-types';

const { Text } = Typography;

export default function DeviceCard({ device, siteId }) {
  const navigate = useNavigate();
  const { id, name, type, status, lastHeartbeat, tagCount } = device;

  const handleClick = () => {
    navigate(`/sites/${siteId}/devices/${id}`);
  };

  return (
    <Card
      hoverable
      onClick={handleClick}
      style={{ borderRadius: 8 }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong>{name}</Text>
        <StatusBadge status={status} />
      </div>
      <div style={{ marginTop: 8, color: '#8c8c8c', fontSize: 13 }}>
        <div>
          <WifiOutlined /> Loại: {type ?? 'Chưa xác định'}
        </div>
        <div>
          <ClockCircleOutlined /> Heartbeat cuối: {lastHeartbeat ? formatRelativeTime(lastHeartbeat) : 'Không có'}
        </div>
        <div>
          <TagOutlined /> Tag: {tagCount ?? 0}
        </div>
      </div>
    </Card>
  );
}

DeviceCard.propTypes = {
  device: PropTypes.shape({
    id: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    type: PropTypes.string,
    status: PropTypes.oneOf(['online', 'offline']),
    lastHeartbeat: PropTypes.string,
    tagCount: PropTypes.number,
  }).isRequired,
  siteId: PropTypes.string.isRequired,
};

// Ghi chú: StatusBadge sẽ được cung cấp trong nhóm components sau. 
// formatRelativeTime là hàm tiện ích sẽ triển khai ở utils/formatters.js.
