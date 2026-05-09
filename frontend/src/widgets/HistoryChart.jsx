import { useState, useEffect, useCallback } from 'react';
import { Select, DatePicker, Button, Spin, Empty } from 'antd';
import { Line } from 'react-chartjs-2';
import zoomPlugin from 'chartjs-plugin-zoom';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import * as historyApi from '../api/history';
import { formatTimestamp } from '../utils/formatters';
import PropTypes from 'prop-types';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
  zoomPlugin
);

const { RangePicker } = DatePicker;

export default function HistoryChart({ siteId, deviceId, availableTags }) {
  const [selectedTags, setSelectedTags] = useState([]);
  const [dateRange, setDateRange] = useState([null, null]);
  const [chartData, setChartData] = useState(null);
  const [loading, setLoading] = useState(false);

  const fetchHistory = useCallback(async () => {
    if (!selectedTags.length || !dateRange[0] || !dateRange[1]) return;
    setLoading(true);
    try {
      const params = {
        tags: selectedTags.join(','),
        from: dateRange[0].toISOString(),
        to: dateRange[1].toISOString(),
      };
      const res = await historyApi.getDeviceData(siteId, deviceId, params);
      // Giả sử response: { timestamps: [], series: { tagName: [] } }
      const { timestamps, series } = res.data;

      const datasets = selectedTags.map((tag, idx) => ({
        label: tag,
        data: series[tag] || [],
        borderColor: `hsl(${(idx * 60) % 360}, 70%, 50%)`,
        backgroundColor: `hsla(${(idx * 60) % 360}, 70%, 50%, 0.1)`,
        fill: true,
        tension: 0.2,
        pointRadius: 0,
      }));

      setChartData({
        labels: timestamps.map(ts => new Date(ts).toLocaleTimeString()),
        datasets,
      });
    } catch (err) {
      console.error('Fetch history error', err);
    } finally {
      setLoading(false);
    }
  }, [siteId, deviceId, selectedTags, dateRange]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      zoom: {
        pan: { enabled: true, mode: 'x' },
        zoom: { wheel: { enabled: true }, pinch: { enabled: true }, mode: 'x' },
      },
    },
    scales: {
      x: { display: true },
      y: { display: true },
    },
  };

  if (!availableTags || availableTags.length === 0) {
    return <Empty description="Không có tag nào để hiển thị lịch sử" />;
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <Select
          mode="multiple"
          placeholder="Chọn tag"
          value={selectedTags}
          onChange={setSelectedTags}
          options={availableTags.map(tag => ({ label: tag.name, value: tag.name }))}
          style={{ minWidth: 200 }}
        />
        <RangePicker
          showTime
          onChange={dates => setDateRange(dates)}
        />
        <Button onClick={fetchHistory} loading={loading}>Xem</Button>
      </div>
      <div style={{ height: 400 }}>
        {loading ? (
          <Spin style={{ marginTop: '20%' }} />
        ) : chartData ? (
          <Line data={chartData} options={chartOptions} />
        ) : (
          <Empty description="Chọn tag và khoảng thời gian để xem dữ liệu" />
        )}
      </div>
    </div>
  );
}

HistoryChart.propTypes = {
  siteId: PropTypes.string.isRequired,
  deviceId: PropTypes.string.isRequired,
  availableTags: PropTypes.array.isRequired,
};