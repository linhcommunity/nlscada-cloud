import { useEffect, useRef, useCallback } from 'react';
import { Alert, Spin, Empty } from 'antd';
import { useWebSocket } from '../hooks/useWebSocket';
import { useConnectionStatus } from '../hooks/useConnectionStatus';
import TagValue from './TagValue';
import OfflineOverlay from '../components/OfflineOverlay';
import PropTypes from 'prop-types';

export default function RealtimePanel({ siteId, deviceId, thresholdsMap }) {
  const { data: tagData, latency, connectionState } = useWebSocket(siteId, deviceId);
  const { isOnline } = useConnectionStatus();

  // tagData có thể là object { [tagName]: { value, unit, timestamp, ... } } hoặc array
  const tags = tagData ? Object.entries(tagData).map(([name, info]) => ({ name, ...info })) : [];

  if (!isOnline || connectionState === 'CLOSED') {
    return (
      <div style={{ position: 'relative', minHeight: 200 }}>
        <OfflineOverlay />
        <Empty description="Mất kết nối dữ liệu" />
      </div>
    );
  }

  if (connectionState === 'CONNECTING') {
    return <Spin tip="Đang kết nối..." style={{ display: 'block', margin: '40px auto' }} />;
  }

  if (!tags.length) {
    return <Empty description="Đang chờ dữ liệu..." />;
  }

  return (
    <div style={{ position: 'relative' }}>
      <div style={{ marginBottom: 12, display: 'flex', gap: 16 }}>
        <span>🔗 Latency: <strong>{latency}ms</strong></span>
        <span>📊 Cập nhật: {new Date().toLocaleTimeString()}</span>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 12 }}>
        {tags.map(tag => (
          <TagValue key={tag.name} tag={tag} thresholds={thresholdsMap?.[tag.id]} />
        ))}
      </div>
    </div>
  );
}

RealtimePanel.propTypes = {
  siteId: PropTypes.string.isRequired,
  deviceId: PropTypes.string.isRequired,
  thresholdsMap: PropTypes.object,
};

//Panel trung tâm giám sát thời gian thực. Sử dụng useWebSocket để nhận dữ liệu, hiển thị danh sách TagValue, có overlay offline.
//Lưu ý: useWebSocket và useConnectionStatus sẽ được cài đặt ở bước sau. Bạn có thể tạm thời mock chúng để test.