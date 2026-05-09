import { useState, useEffect } from 'react';
import { wsClient } from '../ws/wsClient';

export function useConnectionStatus() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [wsState, setWsState] = useState(wsClient.getState());
  const [latency, setLatency] = useState(0); // có thể lấy từ useWebSocket hoặc cập nhật riêng

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    const handleWsChange = (state) => {
      setWsState(state);
      if (state === 'CLOSED' || state === 'RECONNECTING') {
        setIsOnline(false); // xem như mất kết nối dữ liệu
      } else if (state === 'OPEN') {
        setIsOnline(navigator.onLine); // phụ thuộc vào internet
      }
    };
    wsClient.onStateChange(handleWsChange);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      wsClient.offStateChange(handleWsChange);
    };
  }, []);

  // Nếu muốn cập nhật latency từ WS, ta có thể lắng nghe thêm từ wsClient (không bắt buộc ở hook này)
  // Hoặc sử dụng giá trị latency từ useWebSocket và truyền qua context. Tạm thời giữ 0.

  return { isOnline, wsState, latency };
}