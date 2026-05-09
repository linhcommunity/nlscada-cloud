import { Card, Typography } from 'antd';
import { useEffect, useState } from 'react';
import TrendSparkline from './TrendSparkline';
import PropTypes from 'prop-types';

const { Text } = Typography;
//Hiển thị giá trị thời gian thực của một tag. Hỗ trợ màu nền dựa trên ngưỡng (nếu có), kèm đơn vị và timestamp.
export default function TagValue({ tag, thresholds }) {
  const { name, value, unit, timestamp } = tag;
  const [bgColor, setBgColor] = useState('transparent');

  // Xác định màu nền dựa trên ngưỡng (nếu có cấu hình)
  useEffect(() => {
    if (thresholds) {
      const numericValue = parseFloat(value);
      if (!isNaN(numericValue)) {
        if (thresholds.critical && (numericValue >= thresholds.critical.high || numericValue <= thresholds.critical.low)) {
          setBgColor('#fff1f0'); // đỏ nhạt
        } else if (thresholds.warning && (numericValue >= thresholds.warning.high || numericValue <= thresholds.warning.low)) {
          setBgColor('#fff7e6'); // vàng nhạt
        } else {
          setBgColor('transparent');
        }
      }
    }
  }, [value, thresholds]);

  // Sparkline nếu có dữ liệu lịch sử gần đây (mảng recentValues)
  const sparklineData = tag.recentValues || [];

  return (
    <Card
      size="small"
      style={{ background: bgColor, borderRadius: 8, transition: 'background 0.3s' }}
      bodyStyle={{ padding: '12px' }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Text type="secondary" style={{ fontSize: 12 }}>{name}</Text>
          <div style={{ fontSize: 20, fontWeight: 600 }}>
            {value ?? '--'} <Text style={{ fontSize: 14 }}>{unit}</Text>
          </div>
          <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
            {timestamp ? new Date(timestamp).toLocaleTimeString() : ''}
          </Text>
        </div>
        {sparklineData.length > 0 && (
          <TrendSparkline data={sparklineData} width={80} height={30} />
        )}
      </div>
    </Card>
  );
}

TagValue.propTypes = {
  tag: PropTypes.shape({
    name: PropTypes.string.isRequired,
    value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    unit: PropTypes.string,
    timestamp: PropTypes.string,
    recentValues: PropTypes.arrayOf(PropTypes.number),
  }).isRequired,
  thresholds: PropTypes.shape({
    warning: PropTypes.shape({ high: PropTypes.number, low: PropTypes.number }),
    critical: PropTypes.shape({ high: PropTypes.number, low: PropTypes.number }),
  }),
};