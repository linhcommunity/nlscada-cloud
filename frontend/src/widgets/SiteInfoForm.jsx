import { useState, useEffect } from 'react';
import { Form, Input, Button, message, Spin } from 'antd';
import * as sitesApi from '../api/sites';
import { useAuth } from '../store/AuthContext';
import PropTypes from 'prop-types';

export default function SiteInfoForm({ siteId }) {
  const { currentSite, fetchSites } = useAuth();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  const isEditable = currentSite?.role === 'admin' || currentSite?.role === 'operator';

  useEffect(() => {
    if (!siteId) return;
    setLoading(true);
    sitesApi.getSites() // tái sử dụng để lấy chi tiết (nếu có endpoint riêng thì thay)
      .then(res => {
        const site = (res.data || []).find(s => s.id === siteId);
        if (site) {
          form.setFieldsValue({ name: site.name, description: site.description || '' });
        }
      })
      .catch(() => message.error('Không thể tải thông tin site'))
      .finally(() => setLoading(false));
  }, [siteId, form]);

  const handleSave = async (values) => {
    setSaving(true);
    try {
      await sitesApi.updateSite(siteId, values);
      message.success('Cập nhật site thành công');
      fetchSites(); // refresh danh sách site trong context
    } catch (err) {
      message.error('Cập nhật thất bại');
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Spin />;

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} disabled={!isEditable}>
      <Form.Item name="name" label="Tên site" rules={[{ required: true, message: 'Nhập tên site' }]}>
        <Input />
      </Form.Item>
      <Form.Item name="description" label="Mô tả">
        <Input.TextArea rows={3} />
      </Form.Item>
      {isEditable && (
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>
            Lưu thay đổi
          </Button>
        </Form.Item>
      )}
    </Form>
  );
}

SiteInfoForm.propTypes = {
  siteId: PropTypes.string.isRequired,
};