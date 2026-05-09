import { Modal, Input, Space, Typography } from 'antd';
import { useState } from 'react';
import PropTypes from 'prop-types';

const { Text } = Typography;

export default function ConfirmDialog({
  open,
  onOk,
  onCancel,
  title = 'Xác nhận hành động',
  content = 'Bạn có chắc chắn muốn thực hiện hành động này?',
  risk = '',
  confirmText = 'CONFIRM',
  loading = false,
}) {
  const [input, setInput] = useState('');

  const handleOk = () => {
    if (input === confirmText) {
      onOk?.();
      setInput('');
    }
  };

  const handleCancel = () => {
    setInput('');
    onCancel?.();
  };

  return (
    <Modal
      open={open}
      title={title}
      onOk={handleOk}
      onCancel={handleCancel}
      okButtonProps={{
        disabled: input !== confirmText,
        danger: true,
        loading,
      }}
      cancelText="Hủy"
      okText="Xác nhận"
      destroyOnClose
    >
      <Space direction="vertical" style={{ width: '100%' }}>
        <Text>{content}</Text>
        {risk && (
          <Text type="danger">
            <strong>Rủi ro:</strong> {risk}
          </Text>
        )}
        <Text>
          Nhập <Text code>{confirmText}</Text> để xác nhận:
        </Text>
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder={confirmText}
        />
      </Space>
    </Modal>
  );
}

ConfirmDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onOk: PropTypes.func,
  onCancel: PropTypes.func,
  title: PropTypes.string,
  content: PropTypes.string,
  risk: PropTypes.string,
  confirmText: PropTypes.string,
  loading: PropTypes.bool,
};

// Hộp thoại xác nhận 2 bước dành cho các hành động nguy hiểm (điều khiển thiết bị, xóa…). Yêu cầu người dùng nhập từ khóa xác nhận.