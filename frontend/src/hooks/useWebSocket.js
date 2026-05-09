import { useEffect, useState, useCallback, useRef } from 'react';
import { wsClient } from '../ws/wsClient'; // singleton


//Hook quản lý việc subscribe/unsubscribe WebSocket cho một thiết bị cụ thể khi component mount/unmount.
/**
 * useWebSocket – theo dõi dữ liệu real-time của một device
 * @param {string} siteId
 * @param {string} deviceId
 * @returns {{ data: object, latency: number, connectionState: string }}
 */
export function useWebSocket(siteId, deviceId) {
  const [data, setData] = useState({});
  const [latency, setLatency] = useState(0);
  const [connectionState, setConnectionState] = useState(wsClient.getState());
  const dataRef = useRef(data);

  // Lắng nghe thay đổi trạng thái kết nối
  useEffect(() => {
    const handleStateChange = (state) => {
      setConnectionState(state);
    };
    wsClient.onStateChange(handleStateChange);
    return () => wsClient.offStateChange(handleStateChange);
  }, []);

  // Subscribe khi có siteId và deviceId
  useEffect(() => {
    if (!siteId || !deviceId) return;

    // Hàm xử lý dữ liệu tag_update
    const handleTagUpdate = (msg) => {
      if (msg.site_id === siteId && msg.device_id === deviceId) {
        // Cập nhật dữ liệu
        setData(prev => ({
          ...prev,
          [msg.tag_name]: {
            value: msg.value,
            unit: msg.unit,
            timestamp: msg.timestamp,
          },
        }));
        // Tính latency nếu có server_timestamp
        if (msg.server_timestamp) {
          const now = Date.now();
          const serverTime = new Date(msg.server_timestamp).getTime();
          setLatency(now - serverTime);
        }
      }
    };

    // Đăng ký lắng nghe
    wsClient.on('tag_update', handleTagUpdate);

    // Gửi yêu cầu subscribe
    wsClient.subscribe(siteId, deviceId);

    // Cleanup khi unmount hoặc thay đổi device
    return () => {
      wsClient.unsubscribe(siteId, deviceId);
      wsClient.off('tag_update', handleTagUpdate);
      // Không reset data để tránh nhấp nháy, nhưng có thể giữ lại nếu muốn
    };
  }, [siteId, deviceId]);

  return { data, latency, connectionState };
}