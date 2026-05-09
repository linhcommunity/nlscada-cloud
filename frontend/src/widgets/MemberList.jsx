import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Select, Popconfirm, message, Space } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import * as sitesApi from '../api/sites';
import { useAuth } from '../store/AuthContext';
import ConfirmDialog from '../components/ConfirmDialog';
import PropTypes from 'prop-types';

export default function MemberList({ siteId }) {
  const { currentSite } = useAuth();
  const [members, setMembers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('viewer');

  const isAdmin = currentSite?.role === 'admin';

  const fetchMembers = useCallback(async () => {
    setLoading(true);
    try {
      const res = await sitesApi.getMembers(siteId);
      setMembers(res.data || []);
    } catch (err) {
      message.error('Không thể tải danh sách thành viên');
    } finally {
      setLoading(false);
    }
  }, [siteId]);

  useEffect(() => {
    fetchMembers();
  }, [fetchMembers]);

  const handleInvite = async () => {
    if (!inviteEmail) return;
    try {
      await sitesApi.inviteMember(siteId, inviteEmail, inviteRole);
      message.success('Đã mời thành viên');
      setInviteEmail('');
      fetchMembers();
    } catch (err) {
      message.error(err.response?.data?.error || 'Mời thất bại');
    }
  };

  const handleRoleChange = async (memberId, newRole) => {
    try {
      await sitesApi.updateMemberRole(siteId, memberId, newRole);
      message.success('Đã cập nhật vai trò');
      fetchMembers();
    } catch (err) {
      message.error('Cập nhật vai trò thất bại');
    }
  };

  const handleRemove = async (memberId) => {
    try {
      await sitesApi.removeMember(siteId, memberId);
      message.success('Đã xoá thành viên');
      fetchMembers();
    } catch (err) {
      message.error('Không thể xoá thành viên');
    }
  };

  const columns = [
    { title: 'Email', dataIndex: 'email', key: 'email' },
    {
      title: 'Vai trò',
      dataIndex: 'role',
      key: 'role',
      render: (role, record) => {
        if (!isAdmin) return role;
        return (
          <Select
            value={role}
            onChange={(newRole) => handleRoleChange(record.id, newRole)}
            style={{ width: 120 }}
            options={[
              { label: 'Admin', value: 'admin' },
              { label: 'Operator', value: 'operator' },
              { label: 'Viewer', value: 'viewer' },
            ]}
          />
        );
      },
    },
    ...(isAdmin
      ? [
          {
            title: 'Thao tác',
            key: 'action',
            render: (_, record) => (
              <Popconfirm
                title="Xoá thành viên này?"
                onConfirm={() => handleRemove(record.id)}
                okText="Xoá"
                cancelText="Huỷ"
              >
                <Button danger icon={<DeleteOutlined />} size="small" />
              </Popconfirm>
            ),
          },
        ]
      : []),
  ];

  return (
    <div>
      {isAdmin && (
        <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
          <input
            type="email"
            placeholder="Email thành viên"
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            style={{ flex: 1, padding: '4px 8px', borderRadius: 6, border: '1px solid #d9d9d9' }}
          />
          <Select
            value={inviteRole}
            onChange={setInviteRole}
            options={[
              { label: 'Admin', value: 'admin' },
              { label: 'Operator', value: 'operator' },
              { label: 'Viewer', value: 'viewer' },
            ]}
            style={{ width: 120 }}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={handleInvite}>
            Mời
          </Button>
        </div>
      )}
      <Table
        dataSource={members}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={false}
      />
    </div>
  );
}

MemberList.propTypes = {
  siteId: PropTypes.string.isRequired,
};

//Bảng quản lý thành viên trong site. Cho phép admin mời thành viên, thay đổi role, xóa.