import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { Link } from 'react-router-dom';
import { useAuth } from '../store/AuthContext';

const { Title } = Typography;

export default function LoginPage() {
  const { login } = useAuth();
  const [loading, setLoading] = useState(false);

  const onFinish = async (values) => {
    setLoading(true);
    try {
      await login(values.email, values.password);
    } catch (err) {
      message.error(err.response?.data?.error || 'Đăng nhập thất bại');
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f0f2f5' }}>
      <Card style={{ width: 400 }}>
        <Title level={3} style={{ textAlign: 'center' }}>Đăng nhập NL SCADA</Title>
        <Form layout="vertical" onFinish={onFinish} autoComplete="off">
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email', message: 'Email không hợp lệ' }]}>
            <Input placeholder="email@example.com" />
          </Form.Item>
          <Form.Item name="password" label="Mật khẩu" rules={[{ required: true, message: 'Nhập mật khẩu' }]}>
            <Input.Password placeholder="Mật khẩu" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              Đăng nhập
            </Button>
          </Form.Item>
        </Form>
        <div style={{ textAlign: 'center' }}>
          Chưa có tài khoản? <Link to="/register">Đăng ký</Link>
        </div>
      </Card>
    </div>
  );
}