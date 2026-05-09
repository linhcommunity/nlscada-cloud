const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:3000/v1/ws';
const RECONNECT_DELAYS = [1000, 2000, 4000, 8000, 16000, 30000]; // exponential backoff
const PING_INTERVAL = 30000; // 30s
const PONG_TIMEOUT = 10000; // 10s

class WSClient {
  constructor() {
    this.ws = null;
    this.state = 'CLOSED'; // OPEN, CLOSED, CONNECTING, RECONNECTING
    this.listeners = new Map(); // event -> Set(callback)
    this.stateListeners = new Set();
    this.reconnectAttempt = 0;
    this.reconnectTimer = null;
    this.pingTimer = null;
    this.pongTimeout = null;
    this.pendingSubscriptions = new Map(); // key: `${siteId}:${deviceId}` -> true (cho lần đầu)
    this.token = null;
  }

  connect(token) {
    if (this.ws && (this.state === 'OPEN' || this.state === 'CONNECTING')) return;
    this.token = token;
    this.state = 'CONNECTING';
    this.notifyStateChange();
    this.ws = new WebSocket(WS_URL);

    this.ws.onopen = () => {
      this.state = 'OPEN';
      this.reconnectAttempt = 0;
      this.notifyStateChange();
      // Gửi token qua message đầu tiên
      this.send({ action: 'auth', token: this.token });
      // Gửi lại các subscription đang chờ
      this.pendingSubscriptions.forEach((_, key) => {
        const [siteId, deviceId] = key.split(':');
        this.send({ action: 'sub', site_id: siteId, device_id: deviceId });
      });
      // Bắt đầu ping
      this.startPing();
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        // Nếu là pong, bỏ qua
        if (msg.type === 'pong') {
          this.clearPongTimeout();
          return;
        }
        // Phát cho các listener theo loại
        const eventType = msg.type || 'unknown';
        if (this.listeners.has(eventType)) {
          this.listeners.get(eventType).forEach(cb => cb(msg));
        }
        // Cũng phát cho listener chung (tất cả message)
        if (this.listeners.has('*')) {
          this.listeners.get('*').forEach(cb => cb(msg));
        }
      } catch (e) {
        console.error('WS parse error', e);
      }
    };

    this.ws.onclose = (event) => {
      this.stopPing();
      this.state = 'CLOSED';
      this.notifyStateChange();
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WS error', error);
      // onclose sẽ được gọi ngay sau đó
    };
  }

  send(data) {
    if (this.ws && this.state === 'OPEN') {
      this.ws.send(JSON.stringify(data));
    }
  }

  subscribe(siteId, deviceId) {
    const key = `${siteId}:${deviceId}`;
    if (this.pendingSubscriptions.has(key)) return;
    this.pendingSubscriptions.set(key, true);
    this.send({ action: 'sub', site_id: siteId, device_id: deviceId });
  }

  unsubscribe(siteId, deviceId) {
    const key = `${siteId}:${deviceId}`;
    this.pendingSubscriptions.delete(key);
    this.send({ action: 'unsub', site_id: siteId, device_id: deviceId });
  }

  on(eventType, callback) {
    if (!this.listeners.has(eventType)) {
      this.listeners.set(eventType, new Set());
    }
    this.listeners.get(eventType).add(callback);
  }

  off(eventType, callback) {
    if (this.listeners.has(eventType)) {
      this.listeners.get(eventType).delete(callback);
    }
  }

  onStateChange(cb) {
    this.stateListeners.add(cb);
  }

  offStateChange(cb) {
    this.stateListeners.delete(cb);
  }

  getState() {
    return this.state;
  }

  disconnect() {
    this.stopPing();
    this.clearReconnect();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.state = 'CLOSED';
    this.notifyStateChange();
  }

  // Private
  scheduleReconnect() {
    if (this.reconnectAttempt >= RECONNECT_DELAYS.length) {
      this.reconnectAttempt = RECONNECT_DELAYS.length - 1; // keep max
    }
    const delay = RECONNECT_DELAYS[this.reconnectAttempt];
    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempt++;
      this.connect(this.token);
    }, delay);
    this.state = 'RECONNECTING';
    this.notifyStateChange();
  }

  clearReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  startPing() {
    this.pingTimer = setInterval(() => {
      this.send({ type: 'ping' });
      this.pongTimeout = setTimeout(() => {
        console.warn('Pong timeout, closing connection');
        this.ws.close();
      }, PONG_TIMEOUT);
    }, PING_INTERVAL);
  }

  stopPing() {
    if (this.pingTimer) clearInterval(this.pingTimer);
    this.clearPongTimeout();
  }

  clearPongTimeout() {
    if (this.pongTimeout) {
      clearTimeout(this.pongTimeout);
      this.pongTimeout = null;
    }
  }

  notifyStateChange() {
    this.stateListeners.forEach(cb => cb(this.state));
  }
}

// Singleton
export const wsClient = new WSClient();

// Lưu ý: Đảm bảo rằng khi ứng dụng khởi tạo và có token, bạn gọi wsClient.connect(token) ở đâu đó (ví dụ trong AuthContext khi login thành công). Sẽ được thêm vào sau.