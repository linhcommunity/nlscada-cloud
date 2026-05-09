import { Row, Col, Input, Select, Empty, Spin } from 'antd';
import { useState, useMemo } from 'react';
import TagValue from './TagValue';
import PropTypes from 'prop-types';

const { Search } = Input;

export default function TagList({ tags, thresholdsMap, loading }) {
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState('name');

  const filteredTags = useMemo(() => {
    if (!tags) return [];
    let result = [...tags];
    if (search) {
      result = result.filter(tag => tag.name.toLowerCase().includes(search.toLowerCase()));
    }
    if (sortBy === 'name') {
      result.sort((a, b) => a.name.localeCompare(b.name));
    }
    // Thêm các kiểu sắp xếp khác nếu cần
    return result;
  }, [tags, search, sortBy]);

  if (loading) return <Spin style={{ display: 'block', margin: '20px auto' }} />;
  if (!tags || tags.length === 0) return <Empty description="Không có tag nào" />;

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
        <Search
          placeholder="Tìm tag..."
          allowClear
          onChange={e => setSearch(e.target.value)}
          style={{ width: 200 }}
        />
        <Select
          value={sortBy}
          onChange={setSortBy}
          options={[
            { label: 'Tên', value: 'name' },
          ]}
          style={{ width: 120 }}
        />
      </div>
      <Row gutter={[12, 12]}>
        {filteredTags.map(tag => (
          <Col key={tag.id || tag.name} xs={24} sm={12} md={8} lg={6}>
            <TagValue tag={tag} thresholds={thresholdsMap?.[tag.id]} />
          </Col>
        ))}
      </Row>
    </div>
  );
}

TagList.propTypes = {
  tags: PropTypes.array.isRequired,
  thresholdsMap: PropTypes.object, // map id -> thresholds
  loading: PropTypes.bool,
};