import { BrowserRouter } from 'react-router-dom';
import { ConfigProvider, theme as antTheme } from 'antd';
import { AuthProvider } from './store/AuthContext';
import { ThemeProvider, useTheme } from './store/ThemeContext';
import AppRoutes from './AppRoutes'; // sẽ tạo file riêng cho gọn

function ThemedApp() {
  const { currentTheme } = useTheme();

  return (
    <ConfigProvider
      theme={{
        algorithm:
          currentTheme === 'dark'
            ? antTheme.darkAlgorithm
            : antTheme.defaultAlgorithm,
      }}
    >
      <AppRoutes />
    </ConfigProvider>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <ThemeProvider>
        <AuthProvider>
          <ThemedApp />
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
}