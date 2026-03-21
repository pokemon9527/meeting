import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Input, Modal, Form, message, Card } from 'antd';
import {
  VideoCameraAddOutlined,
  TeamOutlined,
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
      navigate(`/meeting/${meeting.meetingId}`);
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
        <h1>视频会议</h1>
        <p>高效、稳定的多人视频会议系统</p>
        {isAuthenticated && (
          <div className="user-info">
            <span>欢迎, {user?.username}</span>
            <Button onClick={logout}>退出登录</Button>
          </div>
        )}
      </div>

      <div className="home-content">
        <Card className="action-card" hoverable>
          <VideoCameraAddOutlined className="action-icon" />
          <h3>创建会议</h3>
          <p>创建一个新的视频会议</p>
          <Button
            type="primary"
            size="large"
            onClick={() => setCreateModalVisible(true)}
          >
            创建会议
          </Button>
        </Card>

        <Card className="action-card" hoverable>
          <TeamOutlined className="action-icon" />
          <h3>加入会议</h3>
          <p>使用会议号加入会议</p>
          <Button
            type="primary"
            size="large"
            onClick={() => setJoinModalVisible(true)}
          >
            加入会议
          </Button>
        </Card>
      </div>

      <Modal
        title="创建会议"
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        footer={null}
      >
        <Form form={createForm} onFinish={handleCreateMeeting} layout="vertical">
          <Form.Item name="title" label="会议主题">
            <Input placeholder="请输入会议主题" />
          </Form.Item>
          <Form.Item name="password" label="会议密码（可选）">
            <Input.Password placeholder="设置会议密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
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
            <Button type="primary" htmlType="submit" loading={loading} block>
              加入
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Home;
