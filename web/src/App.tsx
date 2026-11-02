import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Vscales from './pages/Vscales';
import Devices from './pages/Devices';
import Topology from './pages/Topology';
import Users from './pages/Users';
import Layout from './components/Layout';

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#5eead4' },
    secondary: { main: '#94a3b8' },
    background: { default: '#0b1220', paper: '#111827' },
  },
  typography: {
    fontFamily: '"IBM Plex Sans", "Segoe UI", sans-serif',
    h4: { fontFamily: '"IBM Plex Mono", monospace', fontWeight: 600 },
    h5: { fontFamily: '"IBM Plex Mono", monospace', fontWeight: 600 },
  },
});

const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AuthProvider>
        <Router>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<PrivateRoute><Layout><Dashboard /></Layout></PrivateRoute>} />
            <Route path="/vscales" element={<PrivateRoute><Layout><Vscales /></Layout></PrivateRoute>} />
            <Route path="/devices" element={<PrivateRoute><Layout><Devices /></Layout></PrivateRoute>} />
            <Route path="/topology" element={<PrivateRoute><Layout><Topology /></Layout></PrivateRoute>} />
            <Route path="/users" element={<PrivateRoute><Layout><Users /></Layout></PrivateRoute>} />
          </Routes>
        </Router>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
