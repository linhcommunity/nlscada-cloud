import { useEffect, useState } from 'react';
import { Row, Col, Button, Spin, Empty, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useParams } from 'react-router-dom';
import { getDevices } from '../api/devices';
import DeviceCard from '../widgets/DeviceCard';
import { useAuth } from '../store/AuthContext';

export default function DeviceList() {
  const { siteId } = useParams();
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(false);
  const { currentSite } = useAuth();
  const canAdd = currentSite?.role === 'admin' || currentSite?.role === 'operator';

  const fetchDevices = async () => {
    setLoading(true);
    try {
      const res = await getDevices(siteId);
      setDevices(res.data || []);
    } catch (err) {
      message.error('Không thể tải danh sách thiết bị');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDevices();
  }, [siteId]);

  const handleAdd = () => {
    // Tạm thời prompt tên thiết bị, sau này dùng modal
    const name = prompt('Tên thiết bị mới:');
    if (!name) return;
    // Gọi API thêm (yêu cầu backend hỗ trợ)
    import('../api/devices').then(({ addDevice }) => {
      addDevice(siteId, { name, type: 'generic' })
        .then(() => {
          message.success('Đã thêm thiết bị');
          fetchDevices();
        })
        .catch(err => message.error('Thêm thất bại'));
    });
  };

  if (loading) return <Spin style={{ display: 'block', margin: '40px auto' }} />;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Danh sách thiết bị</h2>
        {canAdd && (
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            Thêm thiết bị
          </Button>
        )}
      </div>
      {devices.length === 0 ? (
        <Empty description="Chưa có thiết bị nào" />
      ) : (
        <Row gutter={[16, 16]}>
          {devices.map(device => (
            <Col key={device.id} xs={24} sm={12} md={8} lg={6}>
              <DeviceCard device={device} siteId={siteId} />
            </Col>
          ))}
        </Row>
      )}
    </div>
  );
}