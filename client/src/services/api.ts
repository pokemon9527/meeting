import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('refreshToken');
      window.location.href = '/';
    }
    return Promise.reject(error);
  }
);

export const authApi = {
  register: (data: { username: string; email: string; password: string }) =>
    api.post('/auth/register', data),
  login: (data: { email: string; password: string }) =>
    api.post('/auth/login', data),
  refreshToken: (refreshToken: string) =>
    api.post('/auth/refresh', { refreshToken }),
  getCurrentUser: () => api.get('/auth/me'),
};

export const meetingApi = {
  createMeeting: (data: any) => api.post('/meetings', data),
  joinMeeting: (data: { meetingId: string; password?: string }) =>
    api.post('/meetings/join', data),
  getMeeting: (meetingId: string) => api.get(`/meetings/${meetingId}`),
  getMyMeetings: () => api.get('/meetings'),
  updateSettings: (meetingId: string, settings: any) =>
    api.put(`/meetings/${meetingId}/settings`, settings),
  endMeeting: (meetingId: string) =>
    api.post(`/meetings/${meetingId}/end`),
  getParticipants: (meetingId: string) =>
    api.get(`/meetings/${meetingId}/participants`),
  deleteMeeting: (meetingId: string) =>
    api.delete(`/meetings/${meetingId}`),
};

export default api;
