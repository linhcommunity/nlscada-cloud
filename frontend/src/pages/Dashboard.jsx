import { useEffect, useState } from 'react';
import { Row, Col, Button, Card, Typography, Spin, Empty } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useAuth } from '../store/AuthContext';
import { createSite } from '../api/sites';
import SiteCard from '../widgets/SiteCard'; // sẽ làm sau, dùng tạm nội dung inline

const { Title } = Typography;

export default function Dashboard() {
  const { sites, currentSite, fetchSites } = useAuth();
  const [loading, setLoading] = useState(false);

  const handleCreateSite = async () => {
    const name = prompt('Tên site mới:');
    if (!name) return;
    setLoading(true);
    try {
      await createSite(name);
      await fetchSites();
    } catch (err) {
      alert('Tạo site thất bại');
    } finally {
      setLoading(false);
    }
  };

  if (!sites) return <Spin style={{ display: 'block', margin: '40vh auto' }} />;

  return (
    <div>
      <Title level={4}>Danh sách Site</Title>
      {sites.length === 0 ? (
        <Empty description="Bạn chưa có site nào">
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreateSite}>
            Tạo Site đầu tiên
          </Button>
        </Empty>
      ) : (
        <>
          <Button type="dashed" icon={<PlusOutlined />} onClick={handleCreateSite} style={{ marginBottom: 16 }}>
            Tạo Site mới
          </Button>
          <Row gutter={[16, 16]}>
            {sites.map((site) => (
              <Col xs={24} sm={12} md={8} key={site.id}>
                <Card hoverable onClick={() => {/* navigate đến site detail? */}}>
                  <p><strong>{site.name}</strong></p>
                  <p>Vai trò: {site.role}</p>
                </Card>
              </Col>
            ))}
          </Row>
        </>
      )}
    </div>
  );
}