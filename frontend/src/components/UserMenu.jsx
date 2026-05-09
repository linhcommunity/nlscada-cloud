import { Avatar, Dropdown, Space, Typography } from 'antd';
import { UserOutlined, LogoutOutlined, SettingOutlined } from '@ant-design/icons';
import { useAuth } from '../store/AuthContext';

const { Text } = Typography;

export default function UserMenu() {
  const { user, logout } = useAuth();
  const email = user?.email || 'Unknown';
  const name = user?.name || email.split('@')[0];

  const items = [
    {
      key: 'profile',
      icon: <SettingOutlined />,
      label: 'Thông tin cá nhân',
      disabled: true, // tương lai
    },
    { type: 'divider' },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Đăng xuất',
      onClick: () => logout(),
    },
  ];

  return (
    <Dropdown menu={{ items }} trigger={['click']}>
      <Space style={{ cursor: 'pointer' }}>
        <Avatar icon={<UserOutlined />} />
        <Text>{name}</Text>
      </Space>
    </Dropdown>
  );
}