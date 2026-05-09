import { Outlet } from 'react-router-dom';
import { Layout as AntLayout } from 'antd';
import Header from './Header';
import Sidebar from './Sidebar';
import OfflineOverlay from './OfflineOverlay';
import { useConnectionStatus } from '../hooks/useConnectionStatus'; // sẽ triển khai sau, tạm mock

const { Content, Sider } = AntLayout;

export default function Layout() {
  // Tạm bỏ qua connection status để code chạy được, sẽ thêm sau
  // const { isOnline } = useConnectionStatus();

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Header />
      <AntLayout>
        <Sider
          breakpoint="lg"
          collapsedWidth="0"
          zeroWidthTriggerStyle={{ top: 0 }}
          style={{ background: 'inherit' }}
        >
          <Sidebar />
        </Sider>
        <Content style={{ margin: '24px', padding: 24, background: 'inherit' }}>
          {/* OfflineOverlay sẽ được thêm sau */}
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  );
}