import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Form, Input, Button, message, Tabs, Card } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, HomeOutlined } from '@ant-design/icons';
import { useUserStore } from '../../stores/userStore';
import './Login.css';

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { login, register, isAuthenticated } = useUserStore();
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('login');

  if (isAuthenticated) {
    navigate('/home');
    return null;
  }

  const handleLogin = async (values: any) => {
    setLoading(true);
    try {
      await login(values.email, values.password);
      message.success('登录成功');
      navigate('/home');
    } catch (error: any) {
      message.error(error.response?.data?.error || error.response?.data?.message || error.message || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (values: any) => {
    setLoading(true);
    try {
      await register(values.username, values.email, values.password);
      message.success('注册成功');
      navigate('/home');
    } catch (error: any) {
      message.error(error.response?.data?.error || error.response?.data?.message || error.message || '注册失败');
    } finally {
      setLoading(false);
    }
  };

  const loginForm = (
    <Form onFinish={handleLogin} size="large">
      <Form.Item
        name="email"
        rules={[
          { required: true, message: '请输入邮箱' },
          { type: 'email', message: '请输入有效邮箱' },
        ]}
      >
        <Input prefix={<MailOutlined />} placeholder="邮箱" />
      </Form.Item>

      <Form.Item
        name="password"
        rules={[{ required: true, message: '请输入密码' }]}
      >
        <Input.Password prefix={<LockOutlined />} placeholder="密码" />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading} block className="submit-btn">
          登录
        </Button>
      </Form.Item>
    </Form>
  );

  const registerForm = (
    <Form onFinish={handleRegister} size="large">
      <Form.Item
        name="username"
        rules={[
          { required: true, message: '请输入用户名' },
          { min: 2, max: 30, message: '用户名长度2-30位' },
        ]}
      >
        <Input prefix={<UserOutlined />} placeholder="用户名" />
      </Form.Item>

      <Form.Item
        name="email"
        rules={[
          { required: true, message: '请输入邮箱' },
          { type: 'email', message: '请输入有效邮箱' },
        ]}
      >
        <Input prefix={<MailOutlined />} placeholder="邮箱" />
      </Form.Item>

      <Form.Item
        name="password"
        rules={[
          { required: true, message: '请输入密码' },
          { min: 6, message: '密码至少6位' },
        ]}
      >
        <Input.Password prefix={<LockOutlined />} placeholder="密码" />
      </Form.Item>

      <Form.Item
        name="confirmPassword"
        dependencies={['password']}
        rules={[
          { required: true, message: '请确认密码' },
          ({ getFieldValue }) => ({
            validator(_, value) {
              if (!value || getFieldValue('password') === value) {
                return Promise.resolve();
              }
              return Promise.reject(new Error('两次密码不一致'));
            },
          }),
        ]}
      >
        <Input.Password prefix={<LockOutlined />} placeholder="确认密码" />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading} block className="submit-btn">
          注册
        </Button>
      </Form.Item>
    </Form>
  );

  return (
    <div className="login-container">
      <div className="login-bg">
        <div className="login-particles" />
      </div>
      
      <Link to="/" className="back-home">
        <HomeOutlined />
        <span>返回首页</span>
      </Link>

      <Card className="login-card">
        <div className="login-header">
          <div className="login-logo">
            <span className="logo-icon">📹</span>
            <span className="logo-text">MeetPro</span>
          </div>
          <p>登录或注册开始使用</p>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          centered
          className="login-tabs"
          items={[
            {
              key: 'login',
              label: '登录',
              children: loginForm,
            },
            {
              key: 'register',
              label: '注册',
              children: registerForm,
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default Login;
