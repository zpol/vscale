import React, { createContext, useContext, useState, useEffect } from 'react';
import { Box, Typography } from '@mui/material';
import { api } from '../api';

interface AuthContextType {
  isAuthenticated: boolean;
  user: any | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within an AuthProvider');
  return context;
};

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const check = async () => {
      const token = localStorage.getItem('token');
      if (token) {
        try {
          const { data } = await api.get('/api/auth/me');
          setUser(data);
          setIsAuthenticated(true);
        } catch {
          logout();
        }
      }
      setLoading(false);
    };
    check();
  }, []);

  const login = async (username: string, password: string) => {
    const { data } = await api.post('/api/auth/login', { username, password });
    if (!data.token) throw new Error('Invalid response');
    localStorage.setItem('token', data.token);
    setIsAuthenticated(true);
    setUser(data.user);
  };

  const logout = () => {
    localStorage.removeItem('token');
    setIsAuthenticated(false);
    setUser(null);
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Typography>Loading session…</Typography>
      </Box>
    );
  }

  return (
    <AuthContext.Provider value={{ isAuthenticated, user, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};
