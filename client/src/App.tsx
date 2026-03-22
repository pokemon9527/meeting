import React, { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import Home from './pages/Home';
import Meeting from './pages/Meeting';
import Login from './pages/Login';
import Landing from './pages/Landing';
import RecordingPlayback from './pages/RecordingPlayback';
import { useUserStore } from './stores/userStore';
import './App.css';

const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useUserStore();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

const App: React.FC = () => {
  const { checkAuth } = useUserStore();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const initAuth = async () => {
      await checkAuth();
      setLoading(false);
    };
    initAuth();
  }, [checkAuth]);

  if (loading) {
    return (
      <div className="app-loading">
        <div className="spinner"></div>
      </div>
    );
  }

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme.darkAlgorithm,
        token: {
          colorPrimary: '#6C5CE7',
          colorBgContainer: '#2D2D44',
          colorBgElevated: '#2D2D44',
          colorBorder: 'rgba(108, 92, 231, 0.3)',
          colorText: '#E2E8F0',
          colorTextSecondary: '#A0AEC0',
          borderRadius: 8,
        },
        components: {
          Button: {
            primaryShadow: '0 4px 20px rgba(108, 92, 231, 0.25)',
          },
        },
      }}
    >
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/login" element={<Login />} />
          <Route
            path="/home"
            element={
              <PrivateRoute>
                <Home />
              </PrivateRoute>
            }
          />
          <Route
            path="/meeting/:meetingId"
            element={
              <PrivateRoute>
                <Meeting />
              </PrivateRoute>
            }
          />
          <Route
            path="/recordings"
            element={
              <PrivateRoute>
                <RecordingPlayback />
              </PrivateRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
