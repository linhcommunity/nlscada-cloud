import axios from 'axios';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/v1',
  headers: { 'Content-Type': 'application/json' },
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('scada-token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Nếu có token mà bị 401, có thể thử refresh (sẽ làm sau), hiện tại logout luôn
      localStorage.removeItem('scada-token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default apiClient;