import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import * as authApi from '../api/auth';
import * as sitesApi from '../api/sites';
import { wsClient } from '../ws/wsClient';

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('scada-token') || null);
  const [sites, setSites] = useState([]);
  const [currentSite, setCurrentSite] = useState(null);
  const navigate = useNavigate();

  const fetchSites = useCallback(async () => {
    if (!token) return;
    try {
      const res = await sitesApi.getSites();
      setSites(res.data || []);
      // Nếu đã có currentSite, cập nhật role mới nhất; ngược lại chọn site đầu tiên
      if (res.data && res.data.length > 0) {
        if (currentSite) {
          const updated = res.data.find(s => s.id === currentSite.id);
          if (updated) setCurrentSite(updated);
          else setCurrentSite(res.data[0]); // site cũ mất quyền
        } else {
          setCurrentSite(res.data[0]);
        }
      } else {
        setCurrentSite(null);
      }
    } catch (err) {
      console.error('Fetch sites failed', err);
      // Nếu lỗi 401 -> logout
      if (err.response?.status === 401) logout();
    }
  }, [token, currentSite?.id]);

  // Khi token thay đổi -> connect/disconnect WebSocket
  useEffect(() => {
    if (token) {
      wsClient.connect(token);
    } else {
      wsClient.disconnect();
    }
    // Cleanup khi unmount
    return () => {
      wsClient.disconnect();
    };
  }, [token]);

  const login = async (email, password) => {
    const res = await authApi.login(email, password);
    const newToken = res.data.token;
    localStorage.setItem('scada-token', newToken);
    setToken(newToken);
    navigate('/');
  };

  const logout = () => {
    localStorage.removeItem('scada-token');
    setToken(null);
    setUser(null);
    setSites([]);
    setCurrentSite(null);
    navigate('/login');
  };

  const changeSite = (site) => {
    setCurrentSite(site);
    // Có thể navigate đến trang thiết bị của site đó nếu đang ở trong context site
  };

  return (
    <AuthContext.Provider
      value={{ user, token, sites, currentSite, login, logout, changeSite, fetchSites }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);