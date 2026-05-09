import { Component } from 'react';
import { Button, Result } from 'antd';

export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <Result
          status="error"
          title="Đã xảy ra lỗi không mong muốn"
          subTitle={this.state.error?.message || 'Vui lòng thử lại sau.'}
          extra={
            <Button type="primary" onClick={this.handleReset}>
              Thử lại
            </Button>
          }
        />
      );
    }

    return this.props.children;
  }
}

// Bắt lỗi render trong toàn bộ cây component. Hiển thị giao diện lỗi thân thiện.