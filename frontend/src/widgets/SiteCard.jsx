import { Card, Tag, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { AppstoreOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import PropTypes from 'prop-types';

const { Text } = Typography;

export default function SiteCard({ site }) {
  const navigate = useNavigate();

  // Giả định site có các trường: id, name, role, deviceCount, activeDeviceCount, status
  const { id, name, role, deviceCount = 0, activeDeviceCount = 0, status } = site;

  const isOnline = status === 'online'; // tạm thời suy từ status, có thể tính từ activeDeviceCount > 0

  return (
    <Card
      hoverable
      onClick={() => navigate(`/sites/${id}`)}
      style={{ borderRadius: 8, borderLeft: `4px solid ${isOnline ? '#52c41a' : '#d9d9d9'}` }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong style={{ fontSize: 16 }}>{name}</Text>
        <Tag color={role === 'admin' ? 'gold' : role === 'operator' ? 'blue' : 'default'}>
          {role}
        </Tag>
      </div>
      <div style={{ marginTop: 12, display: 'flex', gap: 16, color: '#666' }}>
        <span>
          <AppstoreOutlined /> {deviceCount} thiết bị
        </span>
        <span>
          {activeDeviceCount > 0 ? (
            <CheckCircleOutlined style={{ color: '#52c41a' }} />
          ) : (
            <ExclamationCircleOutlined style={{ color: '#faad14' }} />
          )}
          {' '}{activeDeviceCount} hoạt động
        </span>
      </div>
    </Card>
  );
}

SiteCard.propTypes = {
  site: PropTypes.shape({
    id: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    role: PropTypes.string,
    deviceCount: PropTypes.number,
    activeDeviceCount: PropTypes.number,
    status: PropTypes.string,
  }).isRequired,
};