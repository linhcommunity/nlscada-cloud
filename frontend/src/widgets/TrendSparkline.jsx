import { useEffect, useRef } from 'react';
import PropTypes from 'prop-types';

export default function TrendSparkline({ data, width = 80, height = 30, color = '#1890ff' }) {
  const canvasRef = useRef(null);

  useEffect(() => {
    if (!canvasRef.current || !data || data.length < 2) return;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    const w = canvas.width;
    const h = canvas.height;

    // Clear
    ctx.clearRect(0, 0, w, h);

    // Tính min, max
    const maxVal = Math.max(...data);
    const minVal = Math.min(...data);
    const range = maxVal - minVal || 1;

    const xStep = w / (data.length - 1);
    const getY = (val) => h - ((val - minVal) / range) * h;

    ctx.beginPath();
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.5;

    data.forEach((val, i) => {
      const x = i * xStep;
      const y = getY(val);
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });

    ctx.stroke();
  }, [data, width, height, color]);

  return (
    <canvas
      ref={canvasRef}
      width={width}
      height={height}
      style={{ display: 'block' }}
    />
  );
}

TrendSparkline.propTypes = {
  data: PropTypes.arrayOf(PropTypes.number).isRequired,
  width: PropTypes.number,
  height: PropTypes.number,
  color: PropTypes.string,
};

//Biểu đồ tia nhỏ (sparkline) hiển thị xu hướng gần đây của một tag. Dùng canvas thuần để tối ưu hiệu năng.