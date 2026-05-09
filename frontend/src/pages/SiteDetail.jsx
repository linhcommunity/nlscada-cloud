import { Tabs } from 'antd';
import { useParams } from 'react-router-dom';
import SiteInfoForm from '../widgets/SiteInfoForm';
import MemberList from '../widgets/MemberList';
import DeviceList from '../pages/DeviceList'; // hoặc tạo inline nếu muốn

export default function SiteDetail() {
  const { siteId } = useParams();

  const items = [
    {
      key: 'info',
      label: 'Thông tin',
      children: <SiteInfoForm siteId={siteId} />,
    },
    {
      key: 'members',
      label: 'Thành viên',
      children: <MemberList siteId={siteId} />,
    },
    {
      key: 'devices',
      label: 'Thiết bị',
      children: <DeviceList />,
    },
  ];

  return (
    <div>
      <Tabs items={items} />
    </div>
  );
}

//Lưu ý: Việc nhúng DeviceList vào tab giúp trải nghiệm liền mạch. Có thể tách riêng route /sites/:siteId/devices vẫn hoạt động (sẽ thêm route bên dưới). Tuy nhiên trong SiteDetail, ta import trực tiếp component DeviceList để dùng lại.