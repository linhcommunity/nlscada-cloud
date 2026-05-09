/**
 * Định dạng thời gian tương đối (ví dụ: "3 phút trước", "1 giờ trước")
 * @param {string|Date} dateInput
 * @returns {string}
 */
export function formatRelativeTime(dateInput) {
  if (!dateInput) return '';
  const date = new Date(dateInput);
  const now = new Date();
  const diffMs = now - date;
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return `${diffSec} giây trước`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} phút trước`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr} giờ trước`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 30) return `${diffDay} ngày trước`;
  return date.toLocaleDateString();
}

/**
 * Định dạng timestamp thành chuỗi ngày giờ địa phương
 * @param {string|number} timestamp
 * @returns {string}
 */
export function formatTimestamp(timestamp) {
  if (!timestamp) return '';
  return new Date(timestamp).toLocaleString();
}

/**
 * Định dạng số thành chuỗi có dấu phẩy hoặc đơn vị
 * @param {number} value
 * @param {number} decimals
 * @returns {string}
 */
export function formatNumber(value, decimals = 2) {
  if (value == null) return '--';
  return Number(value).toFixed(decimals);
}

/**
 * Rút gọn số lớn (1000 -> 1K, 1000000 -> 1M)
 * @param {number} num
 * @returns {string}
 */
export function compactNumber(num) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
  return num.toString();
}

// Các hàm định dạng thời gian, số liệu.