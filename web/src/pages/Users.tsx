import React, { useEffect, useState } from 'react';
import {
  Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle,
  IconButton, MenuItem, Stack, Table, TableBody, TableCell, TableHead, TableRow,
  TextField, Typography,
} from '@mui/material';
import { Add, Delete } from '@mui/icons-material';
import { api } from '../api';

const Users: React.FC = () => {
  const [users, setUsers] = useState<any[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'member' });

  const load = () => api.get('/api/users').then((r) => setUsers(r.data)).catch((e) => setError(e.response?.data?.error || e.message));

  useEffect(() => { load(); }, []);

  const create = async () => {
    try {
      await api.post('/api/users', form);
      setOpen(false);
      setForm({ username: '', email: '', password: '', role: 'member' });
      load();
    } catch (e: any) {
      setError(e.response?.data?.error || e.message);
    }
  };

  const remove = async (id: number) => {
    if (!window.confirm('Delete user?')) return;
    await api.delete(`/api/users/${id}`);
    load();
  };

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" mb={2}>
        <Typography variant="h5">Users</Typography>
        <Button startIcon={<Add />} variant="contained" onClick={() => setOpen(true)}>Add user</Button>
      </Stack>
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Username</TableCell>
            <TableCell>Email</TableCell>
            <TableCell>Role</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {users.map((u) => (
            <TableRow key={u.ID}>
              <TableCell>{u.username}</TableCell>
              <TableCell>{u.email}</TableCell>
              <TableCell>{u.role}</TableCell>
              <TableCell align="right">
                <IconButton color="error" onClick={() => remove(u.ID)}><Delete /></IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create user</DialogTitle>
        <DialogContent>
          <TextField fullWidth margin="dense" label="Username" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
          <TextField fullWidth margin="dense" label="Email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          <TextField fullWidth margin="dense" type="password" label="Password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
          <TextField select fullWidth margin="dense" label="Role" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
            <MenuItem value="admin">admin</MenuItem>
            <MenuItem value="operator">operator</MenuItem>
            <MenuItem value="member">member</MenuItem>
          </TextField>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={create}>Create</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Users;
