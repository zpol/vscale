import React, { useEffect, useState } from 'react';
import {
  Alert, Box, Chip, FormControl, InputLabel, MenuItem, Select, Table, TableBody,
  TableCell, TableHead, TableRow, Typography, IconButton,
} from '@mui/material';
import { Delete } from '@mui/icons-material';
import { api } from '../api';

const Devices: React.FC = () => {
  const [vscales, setVscales] = useState<any[]>([]);
  const [vscaleId, setVscaleId] = useState<number | ''>('');
  const [devices, setDevices] = useState<any[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.get('/api/vscales').then((r) => {
      setVscales(r.data);
      if (r.data.length) setVscaleId(r.data[0].ID);
    });
  }, []);

  useEffect(() => {
    if (!vscaleId) return;
    const load = () => api.get(`/api/vscales/${vscaleId}/devices`)
      .then((r) => setDevices(r.data))
      .catch((e) => setError(e.message));
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, [vscaleId]);

  const revoke = async (id: number) => {
    if (!window.confirm('Revoke device?')) return;
    await api.delete(`/api/vscales/${vscaleId}/devices/${id}`);
    setDevices((d) => d.filter((x) => x.ID !== id));
  };

  const displayName = (d: any) => d.hostname || d.name || `device-${d.ID}`;

  return (
    <Box>
      <Typography variant="h5" gutterBottom>Devices</Typography>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <FormControl sx={{ minWidth: 240, mb: 2 }} size="small">
        <InputLabel>Vscale</InputLabel>
        <Select label="Vscale" value={vscaleId} onChange={(e) => setVscaleId(e.target.value as number)}>
          {vscales.map((v) => <MenuItem key={v.ID} value={v.ID}>{v.name} ({v.overlayCidr})</MenuItem>)}
        </Select>
      </FormControl>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Hostname</TableCell>
            <TableCell>OS</TableCell>
            <TableCell>Address</TableCell>
            <TableCell>Online</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Last heartbeat</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {devices.map((d) => (
            <TableRow key={d.ID}>
              <TableCell sx={{ fontFamily: 'IBM Plex Mono, monospace', fontWeight: 600 }}>
                {displayName(d)}
              </TableCell>
              <TableCell>{d.os || '—'}</TableCell>
              <TableCell><code>{d.address}</code></TableCell>
              <TableCell>
                <Chip
                  size="small"
                  label={d.online ? 'online' : 'offline'}
                  sx={{
                    fontWeight: 600,
                    bgcolor: d.online ? 'rgba(56,189,248,0.15)' : 'rgba(100,116,139,0.2)',
                    color: d.online ? '#38bdf8' : '#94a3b8',
                    border: `1px solid ${d.online ? '#38bdf8' : '#64748b'}`,
                  }}
                />
              </TableCell>
              <TableCell>{d.status}</TableCell>
              <TableCell>{d.lastHeartbeat ? new Date(d.lastHeartbeat).toLocaleString() : '—'}</TableCell>
              <TableCell align="right">
                <IconButton color="error" onClick={() => revoke(d.ID)}><Delete /></IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );
};

export default Devices;
