import { Button } from 'antd';
import { SunOutlined, MoonOutlined } from '@ant-design/icons';
import { useTheme } from '../store/ThemeContext';

export default function ThemeToggle() {
  const { currentTheme, toggleTheme } = useTheme();
  const isDark = currentTheme === 'dark';

  return (
    <Button
      type="text"
      icon={isDark ? <SunOutlined /> : <MoonOutlined />}
      onClick={toggleTheme}
      title={isDark ? 'Chế độ sáng' : 'Chế độ tối'}
    />
  );
}