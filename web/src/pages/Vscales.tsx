import React, { useEffect, useState } from 'react';
import {
  Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle,
  IconButton, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, Typography,
} from '@mui/material';
import { Add, ContentCopy, Delete, Hub } from '@mui/icons-material';
import { api } from '../api';

interface Vscale {
  ID: number;
  name: string;
  description: string;
  overlayCidr: string;
  dns: string;
  status: string;
  hubs?: any[];
  devices?: any[];
}

const Vscales: React.FC = () => {
  const [list, setList] = useState<Vscale[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [hubOpen, setHubOpen] = useState<Vscale | null>(null);
  const [hubName, setHubName] = useState('hub-1');
  const [hubEndpoint, setHubEndpoint] = useState('');
  const [hubToken, setHubToken] = useState<string | null>(null);
  const [tokenOpen, setTokenOpen] = useState<Vscale | null>(null);
  const [joinToken, setJoinToken] = useState<string | null>(null);

  const load = () => api.get('/api/vscales').then((r) => setList(r.data)).catch((e) => setError(e.message));

  useEffect(() => { load(); }, []);

  const create = async () => {
    try {
      await api.post('/api/vscales', { name, description: desc });
      setOpen(false);
      setName('');
      setDesc('');
      load();
    } catch (e: any) {
      setError(e.response?.data?.error || e.message);
    }
  };

  const remove = async (id: number) => {
    if (!window.confirm('Delete vscale and all hubs/devices?')) return;
    await api.delete(`/api/vscales/${id}`);
    load();
  };

  const createHub = async () => {
    if (!hubOpen) return;
    try {
      const { data } = await api.post(`/api/vscales/${hubOpen.ID}/hubs`, {
        name: hubName,
        endpoint: hubEndpoint,
        listenPort: 51820,
      });
      setHubToken(data.agentToken);
      load();
    } catch (e: any) {
      setError(e.response?.data?.error || e.message);
    }
  };

  const createJoin = async () => {
    if (!tokenOpen) return;
    const { data } = await api.post(`/api/vscales/${tokenOpen.ID}/join-tokens`, {
      label: 'agent',
      reusable: true,
      uses: 100,
      expiresInHours: 168,
    });
    setJoinToken(data.token);
  };

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5">Vscales</Typography>
        <Button startIcon={<Add />} variant="contained" onClick={() => setOpen(true)}>Create</Button>
      </Stack>
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Overlay</TableCell>
            <TableCell>DNS</TableCell>
            <TableCell>Hubs</TableCell>
            <TableCell>Devices</TableCell>
            <TableCell>Status</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {list.map((v) => (
            <TableRow key={v.ID}>
              <TableCell>{v.name}</TableCell>
              <TableCell><code>{v.overlayCidr}</code></TableCell>
              <TableCell>{v.dns}</TableCell>
              <TableCell>{v.hubs?.length ?? 0}</TableCell>
              <TableCell>{v.devices?.length ?? 0}</TableCell>
              <TableCell><Chip size="small" label={v.status} color={v.status === 'active' ? 'success' : 'default'} /></TableCell>
              <TableCell align="right">
                <IconButton title="Add hub" onClick={() => { setHubOpen(v); setHubToken(null); }}><Hub /></IconButton>
                <IconButton title="Join token" onClick={() => { setTokenOpen(v); setJoinToken(null); }}><ContentCopy /></IconButton>
                <IconButton color="error" onClick={() => remove(v.ID)}><Delete /></IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create vscale</DialogTitle>
        <DialogContent>
          <TextField fullWidth margin="dense" label="Name" value={name} onChange={(e) => setName(e.target.value)} />
          <TextField fullWidth margin="dense" label="Description" value={desc} onChange={(e) => setDesc(e.target.value)} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={create}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!hubOpen} onClose={() => setHubOpen(null)} fullWidth maxWidth="sm">
        <DialogTitle>Add hub to {hubOpen?.name}</DialogTitle>
        <DialogContent>
          <TextField fullWidth margin="dense" label="Hub name" value={hubName} onChange={(e) => setHubName(e.target.value)} />
          <TextField fullWidth margin="dense" label="Public endpoint (host or host:port)" value={hubEndpoint} onChange={(e) => setHubEndpoint(e.target.value)}
            helperText="IP/DNS reachable by clients, e.g. 192.168.68.113" />
          {hubToken && (
            <Alert severity="success" sx={{ mt: 2 }}>
              Hub agent token (save it):
              <Box component="pre" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{hubToken}</Box>
              <Typography variant="caption">vscale hub --server {'<PUBLIC_URL>'} --token {hubToken}</Typography>
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setHubOpen(null)}>Close</Button>
          {!hubToken && <Button variant="contained" onClick={createHub}>Create hub</Button>}
        </DialogActions>
      </Dialog>

      <Dialog open={!!tokenOpen} onClose={() => setTokenOpen(null)} fullWidth maxWidth="sm">
        <DialogTitle>Join token — {tokenOpen?.name}</DialogTitle>
        <DialogContent>
          {joinToken ? (
            <Alert severity="info">
              <Box component="pre" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{joinToken}</Box>
              <Typography variant="caption">vscale join {joinToken} --server {'<PUBLIC_URL>'}</Typography>
            </Alert>
          ) : (
            <Typography>Generate a reusable join token for agents.</Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setTokenOpen(null)}>Close</Button>
          {!joinToken && <Button variant="contained" onClick={createJoin}>Generate</Button>}
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Vscales;
