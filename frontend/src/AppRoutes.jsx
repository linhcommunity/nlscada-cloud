import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import Dashboard from './pages/Dashboard';
import SiteDetail from './pages/SiteDetail';
import DeviceList from './pages/DeviceList';
import DeviceDetail from './pages/DeviceDetail';
import ProtectedRoute from './components/ProtectedRoute';

export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="sites/:siteId" element={<SiteDetail />} />
          <Route path="sites/:siteId/devices" element={<DeviceList />} />
          <Route path="sites/:siteId/devices/:deviceId" element={<DeviceDetail />} />
          {/* Có thể thêm route members riêng nếu thích */}
          <Route path="sites/:siteId/members" element={<SiteDetail />} /> 
          {/* Sử dụng SiteDetail, tab Members sẽ active nếu cần; hoặc tạo trang riêng */}
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}