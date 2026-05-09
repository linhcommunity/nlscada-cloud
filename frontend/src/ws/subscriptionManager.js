// ws/subscriptionManager.js
// Hiện tại, singleton wsClient đã quản lý sub/unsub đủ dùng.
// File này có thể dùng để tạo một lớp quản lý reference count nếu cần.

class SubscriptionManager {
  constructor(wsClient) {
    this.wsClient = wsClient;
    this.refCounts = new Map(); // key -> số lượng component đang sub
  }

  subscribe(siteId, deviceId) {
    const key = `${siteId}:${deviceId}`;
    const count = this.refCounts.get(key) || 0;
    if (count === 0) {
      this.wsClient.subscribe(siteId, deviceId);
    }
    this.refCounts.set(key, count + 1);
  }

  unsubscribe(siteId, deviceId) {
    const key = `${siteId}:${deviceId}`;
    const count = this.refCounts.get(key) || 0;
    if (count <= 1) {
      this.wsClient.unsubscribe(siteId, deviceId);
      this.refCounts.delete(key);
    } else if (count > 1) {
      this.refCounts.set(key, count - 1);
    }
  }
}

// Hiện tại chưa dùng, nhưng để sẵn sàng
export const subscriptionManager = new SubscriptionManager(wsClient);

//Quản lý subscription một cách an toàn, tránh trùng lặp. 
// Hiện tại đã được xử lý bên trong wsClient bằng pendingSubscriptions. 
// Tuy nhiên, ta vẫn tạo file này để có thể mở rộng nếu cần quản lý phức tạp hơn 
// (ví dụ nhiều component cùng sub một device, đếm tham chiếu). 
// Trong phiên bản đơn giản, nó có thể re-export hoặc bổ sung logic.