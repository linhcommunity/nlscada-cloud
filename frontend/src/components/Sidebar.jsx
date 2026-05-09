import { Menu } from 'antd';
import { DashboardOutlined, AppstoreOutlined, TeamOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../store/AuthContext';

export default function Sidebar() {
  const { currentSite } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const items = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: 'Dashboard',
    },
    ...(currentSite
      ? [
          {
            key: `/sites/${currentSite.id}`,
            icon: <AppstoreOutlined />,
            label: 'Quản lý Site',
          },
          {
            key: `/sites/${currentSite.id}/devices`,
            icon: <AppstoreOutlined />,
            label: 'Thiết bị',
          },
          {
            key: `/sites/${currentSite.id}/members`,
            icon: <TeamOutlined />,
            label: 'Thành viên',
          },
        ]
      : []),
  ];

  const handleClick = ({ key }) => {
    navigate(key);
  };

  return (
    <Menu
      mode="inline"
      selectedKeys={[location.pathname === '/' ? '/' : location.pathname]}
      onClick={handleClick}
      items={items}
      style={{ height: '100%', borderRight: 0, paddingTop: 16 }}
    />
  );
}