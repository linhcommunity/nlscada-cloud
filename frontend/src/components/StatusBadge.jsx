import { Badge } from 'antd';
import PropTypes from 'prop-types';

export default function StatusBadge({ status }) {
  const isOnline = status === 'online';
  const color = isOnline ? '#52c41a' : '#d9d9d9';

  return (
    <Badge
      status={isOnline ? 'processing' : 'default'}
      color={color}
      text={isOnline ? 'Online' : 'Offline'}
    />
  );
}

StatusBadge.propTypes = {
  status: PropTypes.oneOf(['online', 'offline']),
};