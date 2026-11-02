import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Box, Button, Paper, TextField, Typography } from '@mui/material';
import { useAuth } from '../contexts/AuthContext';

const Login: React.FC = () => {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await login(username, password);
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'radial-gradient(ellipse at top, #134e4a 0%, #0b1220 55%)',
    }}>
      <Paper sx={{ p: 4, width: 380, background: 'rgba(17,24,39,0.92)' }} component="form" onSubmit={submit}>
        <Typography variant="h4" gutterBottom>vscale</Typography>
        <Typography variant="body2" color="text.secondary" mb={3}>Sign in to the control plane</Typography>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <TextField fullWidth margin="dense" label="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
        <TextField fullWidth margin="dense" type="password" label="Password" value={password} onChange={(e) => setPassword(e.target.value)} />
        <Button fullWidth type="submit" variant="contained" sx={{ mt: 2 }} disabled={loading}>
          {loading ? 'Signing in…' : 'Sign in'}
        </Button>
      </Paper>
    </Box>
  );
};

export default Login;
