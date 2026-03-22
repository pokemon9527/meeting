import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Button, Input, Modal, Form, message, Card } from 'antd';
import {
  VideoCameraAddOutlined,
  TeamOutlined,
  HomeOutlined,
  LogoutOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { meetingApi } from '../../services/api';
import { useUserStore } from '../../stores/userStore';
import './Home.css';

const Home: React.FC = () => {
  const navigate = useNavigate();
  const { user, isAuthenticated, logout } = useUserStore();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [joinModalVisible, setJoinModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [createForm] = Form.useForm();
  const [joinForm] = Form.useForm();

  const handleCreateMeeting = async (values: any) => {
    setLoading(true);
    try {
      const response = await meetingApi.createMeeting({
        title: values.title || '视频会议',
        password: values.password || undefined,
      });
      const meeting = response.data.data.meeting;
      message.success('会议创建成功');
      setCreateModalVisible(false);
      createForm.resetFields();
      navigate(`/meeting/${meeting.meeting_id}`);
    } catch (error: any) {
      message.error(error.response?.data?.message || '创建失败');
    } finally {
      setLoading(false);
    }
  };

  const handleJoinMeeting = async (values: any) => {
    setLoading(true);
    try {
      await meetingApi.joinMeeting({
        meetingId: values.meetingId,
        password: values.password || undefined,
      });
      message.success('加入会议成功');
      setJoinModalVisible(false);
      joinForm.resetFields();
      navigate(`/meeting/${values.meetingId}`);
    } catch (error: any) {
      message.error(error.response?.data?.message || '加入失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="home-container">
      <div className="home-header">
        <div className="header-left">
          <Link to="/" className="back-home">
            <HomeOutlined />
            <span>返回首页</span>
          </Link>
        </div>
        <div className="header-center">
          <h1>视频会议</h1>
          <p>欢迎回来，{user?.username}</p>
        </div>
        <div className="header-right">
          <Button 
            icon={<LogoutOutlined />} 
            onClick={logout}
            className="logout-btn"
          >
            退出登录
          </Button>
        </div>
      </div>

      <div className="home-content">
        <Card className="action-card create-card" hoverable>
          <div className="card-icon">
            <VideoCameraAddOutlined />
          </div>
          <h3>创建会议</h3>
          <p>创建一个新的视频会议</p>
          <Button
            type="primary"
            size="large"
            onClick={() => setCreateModalVisible(true)}
            className="action-btn"
          >
            创建会议
          </Button>
        </Card>

        <Card className="action-card join-card" hoverable>
          <div className="card-icon">
            <TeamOutlined />
          </div>
          <h3>加入会议</h3>
          <p>使用会议号加入会议</p>
          <Button
            type="primary"
            size="large"
            onClick={() => setJoinModalVisible(true)}
            className="action-btn"
          >
            加入会议
          </Button>
        </Card>

        <Card className="action-card recordings-card" hoverable>
          <div className="card-icon">
            <VideoCameraOutlined />
          </div>
          <h3>录像回放</h3>
          <p>查看会议录像</p>
          <Button
            type="primary"
            size="large"
            onClick={() => navigate('/recordings')}
            className="action-btn"
          >
            查看录像
          </Button>
        </Card>
      </div>

      <Modal
        title="创建会议"
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        footer={null}
        className="meeting-modal"
      >
        <Form form={createForm} onFinish={handleCreateMeeting} layout="vertical">
          <Form.Item name="title" label="会议主题">
            <Input placeholder="请输入会议主题" />
          </Form.Item>
          <Form.Item name="password" label="会议密码（可选）">
            <Input.Password placeholder="设置会议密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block className="submit-btn">
              创建
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="加入会议"
        open={joinModalVisible}
        onCancel={() => setJoinModalVisible(false)}
        footer={null}
        className="meeting-modal"
      >
        <Form form={joinForm} onFinish={handleJoinMeeting} layout="vertical">
          <Form.Item
            name="meetingId"
            label="会议号"
            rules={[{ required: true, message: '请输入会议号' }]}
          >
            <Input placeholder="请输入6位会议号" maxLength={6} />
          </Form.Item>
          <Form.Item name="password" label="会议密码">
            <Input.Password placeholder="如有密码请输入" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block className="submit-btn">
              加入
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Home;
