import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from 'antd';
import {
  VideoCameraOutlined,
  DesktopOutlined,
  MessageOutlined,
  RecordOutlined,
  SafetyOutlined,
  GlobalOutlined,
  TeamOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import './Landing.css';

const features = [
  {
    icon: <VideoCameraOutlined />,
    title: '高清视频会议',
    description: '支持1080P高清画质，流畅的视频通话体验，SFU架构支持50-200人同时在线',
    color: '#6C5CE7',
  },
  {
    icon: <DesktopOutlined />,
    title: '屏幕共享',
    description: '即时共享屏幕内容，支持选取窗口共享，让演示更灵活高效',
    color: '#00CEC9',
  },
  {
    icon: <MessageOutlined />,
    title: '实时聊天',
    description: '会议中实时文字聊天，支持私聊和群聊，随时沟通无障碍',
    color: '#FDCB6E',
  },
  {
    icon: <RecordOutlined />,
    title: '会议录制',
    description: '云端录制会议内容，多码率转码存储，随时回放查看',
    color: '#E17055',
  },
  {
    icon: <SafetyOutlined />,
    title: '端到端加密',
    description: 'WebRTC DTLS-SRTP加密，保障会议内容安全，放心沟通',
    color: '#00B894',
  },
  {
    icon: <GlobalOutlined />,
    title: '跨平台支持',
    description: '支持Chrome、Firefox、Safari等主流浏览器，随时随地加入会议',
    color: '#74b9ff',
  },
];

const steps = [
  {
    number: '01',
    title: '创建会议',
    description: '点击"开始会议"按钮，快速创建专属会议室',
  },
  {
    number: '02',
    title: '邀请成员',
    description: '复制会议链接或输入会议号，一键邀请参与者加入',
  },
  {
    number: '03',
    title: '开始协作',
    description: '开启视频、屏幕共享，实时互动，高效沟通',
  },
];

const Landing: React.FC = () => {
  const navigate = useNavigate();

  return (
    <div className="landing">
      {/* Navigation */}
      <nav className="landing-nav">
        <div className="nav-container">
          <div className="nav-logo">
            <span className="logo-icon">📹</span>
            <span className="logo-text">MeetPro</span>
          </div>
          <div className="nav-actions">
            <Button type="text" onClick={() => navigate('/login')}>
              登录
            </Button>
            <Button type="primary" onClick={() => navigate('/login')}>
              免费开始
            </Button>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="hero">
        <div className="hero-bg">
          <div className="hero-gradient" />
          <div className="hero-particles" />
        </div>
        <div className="hero-content">
          <div className="hero-badge">
            <span className="badge-dot" />
            全新视频会议体验
          </div>
          <h1 className="hero-title">
            重新定义
            <span className="text-gradient"> 视频会议</span>
          </h1>
          <p className="hero-subtitle">
            流畅、稳定、安全的在线会议平台。支持多人视频通话、屏幕共享、实时聊天和会议录制，让团队协作更高效。
          </p>
          <div className="hero-actions">
            <Button
              type="primary"
              size="large"
              icon={<VideoCameraOutlined />}
              onClick={() => navigate('/login')}
              className="hero-btn-primary"
            >
              免费开始会议
            </Button>
            <Button
              size="large"
              icon={<TeamOutlined />}
              onClick={() => navigate('/login')}
              className="hero-btn-secondary"
            >
              查看演示
            </Button>
          </div>
          <div className="hero-stats">
            <div className="stat-item">
              <span className="stat-number">50+</span>
              <span className="stat-label">人同时在线</span>
            </div>
            <div className="stat-divider" />
            <div className="stat-item">
              <span className="stat-number">1080P</span>
              <span className="stat-label">高清画质</span>
            </div>
            <div className="stat-divider" />
            <div className="stat-item">
              <span className="stat-number">99.9%</span>
              <span className="stat-label">服务可用性</span>
            </div>
          </div>
        </div>
        <div className="hero-visual">
          <div className="video-preview">
            <div className="preview-header">
              <div className="preview-dots">
                <span className="dot dot-red" />
                <span className="dot dot-yellow" />
                <span className="dot dot-green" />
              </div>
              <span className="preview-title">会议进行中</span>
            </div>
            <div className="preview-grid">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="preview-tile">
                  <div className="tile-avatar">
                    <span>{['A', 'B', 'C', 'D'][i - 1]}</span>
                  </div>
                  <div className="tile-name">参与者 {i}</div>
                </div>
              ))}
            </div>
            <div className="preview-controls">
              <span className="control-btn">🎤</span>
              <span className="control-btn">📹</span>
              <span className="control-btn">🖥️</span>
              <span className="control-btn active">📞</span>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="features">
        <div className="section-header">
          <h2 className="section-title">强大的会议功能</h2>
          <p className="section-subtitle">一站式解决团队协作中的所有沟通需求</p>
        </div>
        <div className="features-grid">
          {features.map((feature, index) => (
            <div key={index} className="feature-card" style={{ '--accent-color': feature.color } as React.CSSProperties}>
              <div className="feature-icon" style={{ background: `${feature.color}20`, color: feature.color }}>
                {feature.icon}
              </div>
              <h3 className="feature-title">{feature.title}</h3>
              <p className="feature-desc">{feature.description}</p>
            </div>
          ))}
        </div>
      </section>

      {/* How It Works */}
      <section className="how-it-works">
        <div className="section-header">
          <h2 className="section-title">简单三步，快速开会</h2>
          <p className="section-subtitle">无需下载安装，打开浏览器即可使用</p>
        </div>
        <div className="steps">
          {steps.map((step, index) => (
            <div key={index} className="step">
              <div className="step-number">{step.number}</div>
              <div className="step-content">
                <h3 className="step-title">{step.title}</h3>
                <p className="step-desc">{step.description}</p>
              </div>
              {index < steps.length - 1 && <div className="step-arrow"><ArrowRightOutlined /></div>}
            </div>
          ))}
        </div>
      </section>

      {/* CTA Section */}
      <section className="cta">
        <div className="cta-content">
          <h2 className="cta-title">准备好开始了吗？</h2>
          <p className="cta-subtitle">立即注册，免费使用所有功能</p>
          <div className="cta-features">
            <div className="cta-feature">
              <CheckCircleOutlined /> 免费使用
            </div>
            <div className="cta-feature">
              <CheckCircleOutlined /> 无需下载
            </div>
            <div className="cta-feature">
              <CheckCircleOutlined /> 即开即用
            </div>
          </div>
          <Button
            type="primary"
            size="large"
            icon={<VideoCameraOutlined />}
            onClick={() => navigate('/login')}
            className="cta-btn"
          >
            立即开始
          </Button>
        </div>
      </section>

      {/* Footer */}
      <footer className="landing-footer">
        <div className="footer-content">
          <div className="footer-brand">
            <div className="footer-logo">
              <span className="logo-icon">📹</span>
              <span className="logo-text">MeetPro</span>
            </div>
            <p className="footer-tagline">让视频会议更简单</p>
          </div>
          <div className="footer-links">
            <div className="footer-column">
              <h4>产品</h4>
              <a href="#features">功能介绍</a>
              <a href="#pricing">定价方案</a>
              <a href="#security">安全中心</a>
            </div>
            <div className="footer-column">
              <h4>支持</h4>
              <a href="#help">帮助中心</a>
              <a href="#contact">联系我们</a>
              <a href="#faq">常见问题</a>
            </div>
            <div className="footer-column">
              <h4>法律</h4>
              <a href="#privacy">隐私政策</a>
              <a href="#terms">服务条款</a>
            </div>
          </div>
        </div>
        <div className="footer-bottom">
          <p>&copy; 2026 MeetPro. All rights reserved.</p>
        </div>
      </footer>
    </div>
  );
};

export default Landing;
