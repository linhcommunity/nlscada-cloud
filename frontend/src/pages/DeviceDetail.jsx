import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Tabs, Spin, message } from 'antd';
import { getTags, getDevice } from '../api/devices';
import TagList from '../widgets/TagList';
import RealtimePanel from '../widgets/RealtimePanel';
import HistoryChart from '../widgets/HistoryChart';
import { useAuth } from '../store/AuthContext';

export default function DeviceDetail() {
  const { siteId, deviceId } = useParams();
  const [device, setDevice] = useState(null);
  const [tags, setTags] = useState([]);
  const [loading, setLoading] = useState(false);
  const { currentSite } = useAuth();

  const fetchTags = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getTags(siteId, deviceId);
      setTags(res.data || []);
    } catch (err) {
      message.error('Không thể tải danh sách tag');
    } finally {
      setLoading(false);
    }
  }, [siteId, deviceId]);

  const fetchDevice = async () => {
    try {
      const res = await getDevice(siteId, deviceId);
      setDevice(res.data);
    } catch (err) {
      // silent
    }
  };

  useEffect(() => {
    fetchDevice();
    fetchTags();
  }, [siteId, deviceId, fetchTags]);

  // Chuẩn bị dữ liệu cho HistoryChart
  const availableTags = tags.map(t => ({ name: t.name }));

  const items = [
    {
      key: 'tags',
      label: 'Tags',
      children: (
        <TagList
          tags={tags}
          loading={loading}
          // thresholdsMap chưa có, để trống
        />
      ),
    },
    {
      key: 'realtime',
      label: 'Real-time',
      children: (
        <RealtimePanel
          siteId={siteId}
          deviceId={deviceId}
          thresholdsMap={null} // có thể lấy từ API ngưỡng
        />
      ),
    },
    {
      key: 'history',
      label: 'Lịch sử',
      children: (
        <HistoryChart
          siteId={siteId}
          deviceId={deviceId}
          availableTags={availableTags}
        />
      ),
    },
  ];

  return (
    <div>
      <h2>{device?.name || deviceId}</h2>
      <Tabs items={items} />
    </div>
  );
}