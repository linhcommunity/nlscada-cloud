import { Layout, Space, Button, Typography } from 'antd';
import { SettingOutlined, LogoutOutlined, BellOutlined } from '@ant-design/icons';
import SiteSelector from './SiteSelector';
import ThemeToggle from './ThemeToggle';
import LatencyBadge from './LatencyBadge';
import { useAuth } from '../store/AuthContext';

const { Header: AntHeader } = Layout;
const { Text } = Typography;

export default function Header() {
  const { user, logout } = useAuth();

  return (
    <AntHeader
      style={{
        background: '#fff',
        padding: '0 24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderBottom: '1px solid #f0f0f0',
      }}
    >
      <Space size="large">
        <Text strong style={{ fontSize: 18 }}>NL SCADA</Text>
        <SiteSelector />
      </Space>

      <Space>
        <LatencyBadge />
        <BellOutlined style={{ fontSize: 18 }} />
        <ThemeToggle />
        <Button type="text" icon={<LogoutOutlined />} onClick={logout} />
      </Space>
    </AntHeader>
  );
}