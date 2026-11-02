import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert, Box, Chip, FormControl, InputLabel, MenuItem, Paper, Select, Stack,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography,
} from '@mui/material';
import { api } from '../api';

interface Node {
  id: string;
  type: string;
  label: string;
  hostname?: string;
  address: string;
  status: string;
  os?: string;
  online: boolean;
  cidrs?: string[];
  lastSeen?: string;
}

interface Edge {
  id: string;
  source: string;
  target: string;
  kind: string;
  label?: string;
}

interface Graph {
  vscaleId: number;
  overlayCidr: string;
  nodes: Node[];
  edges: Edge[];
}

const statusChip = (n: Node) => {
  if (n.type === 'hub') {
    return (
      <Chip
        size="small"
        label={n.online ? 'hub online' : 'hub offline'}
        sx={{
          fontWeight: 600,
          bgcolor: n.online ? 'rgba(94,234,212,0.15)' : 'rgba(100,116,139,0.2)',
          color: n.online ? '#5eead4' : '#94a3b8',
          border: `1px solid ${n.online ? '#5eead4' : '#64748b'}`,
        }}
      />
    );
  }
  return (
    <Chip
      size="small"
      label={n.online ? 'online' : 'offline'}
      sx={{
        fontWeight: 600,
        bgcolor: n.online ? 'rgba(56,189,248,0.15)' : 'rgba(100,116,139,0.2)',
        color: n.online ? '#38bdf8' : '#94a3b8',
        border: `1px solid ${n.online ? '#38bdf8' : '#64748b'}`,
      }}
    />
  );
};

const Topology: React.FC = () => {
  const [vscales, setVscales] = useState<any[]>([]);
  const [vscaleId, setVscaleId] = useState<number | ''>('');
  const [graph, setGraph] = useState<Graph | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.get('/api/vscales').then((r) => {
      setVscales(r.data);
      if (r.data.length) setVscaleId(r.data[0].ID);
    });
  }, []);

  useEffect(() => {
    if (!vscaleId) return;
    const load = () => api.get(`/api/vscales/${vscaleId}/topology`)
      .then((r) => setGraph(r.data))
      .catch((e) => setError(e.message));
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, [vscaleId]);

  const layout = useMemo(() => {
    if (!graph) return { positions: {} as Record<string, { x: number; y: number }>, w: 900, h: 420 };
    const hubs = graph.nodes.filter((n) => n.type === 'hub');
    const devices = graph.nodes.filter((n) => n.type === 'device');
    const w = 900;
    const h = 420;
    const positions: Record<string, { x: number; y: number }> = {};
    hubs.forEach((n, i) => {
      positions[n.id] = { x: w / 2 + (i - (hubs.length - 1) / 2) * 180, y: h / 2 };
    });
    devices.forEach((n, i) => {
      const angle = (Math.PI * 2 * i) / Math.max(devices.length, 1) - Math.PI / 2;
      const r = 160;
      positions[n.id] = { x: w / 2 + Math.cos(angle) * r, y: h / 2 + Math.sin(angle) * r };
    });
    return { positions, w, h };
  }, [graph]);

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5">Topology</Typography>
        <FormControl size="small" sx={{ minWidth: 240 }}>
          <InputLabel>Vscale</InputLabel>
          <Select label="Vscale" value={vscaleId} onChange={(e) => setVscaleId(e.target.value as number)}>
            {vscales.map((v) => <MenuItem key={v.ID} value={v.ID}>{v.name}</MenuItem>)}
          </Select>
        </FormControl>
      </Stack>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {graph && (
        <>
          <Paper sx={{ p: 2, overflow: 'auto', mb: 2, background: 'radial-gradient(circle at center,#0f172a,#020617)' }}>
            <Typography variant="body2" color="text.secondary" mb={1}>
              Overlay <code>{graph.overlayCidr}</code> · {graph.nodes.length} nodes ·{' '}
              {graph.edges.filter((e) => e.kind === 'tunnel').length} tunnels
            </Typography>
            <svg width="100%" viewBox={`0 0 ${layout.w} ${layout.h}`} style={{ maxHeight: 420 }}>
              {graph.edges.filter((e) => e.kind === 'tunnel').map((e) => {
                const a = layout.positions[e.source];
                const b = layout.positions[e.target];
                if (!a || !b) return null;
                return <line key={e.id} x1={a.x} y1={a.y} x2={b.x} y2={b.y} stroke="#334155" strokeWidth={2} />;
              })}
              {graph.nodes.map((n) => {
                const p = layout.positions[n.id];
                if (!p) return null;
                const color = n.type === 'hub' ? '#5eead4' : (n.online ? '#38bdf8' : '#64748b');
                const title = n.hostname || n.label;
                return (
                  <g key={n.id} transform={`translate(${p.x},${p.y})`}>
                    <circle r={n.type === 'hub' ? 28 : 20} fill="#111827" stroke={color} strokeWidth={3} />
                    <text textAnchor="middle" y={4} fill="#e2e8f0" fontSize={11} fontFamily="IBM Plex Mono, monospace">
                      {title.slice(0, 12)}
                    </text>
                    <text textAnchor="middle" y={42} fill="#94a3b8" fontSize={10}>{n.address}</text>
                  </g>
                );
              })}
            </svg>
            <Stack direction="row" spacing={1} mt={1} flexWrap="wrap" useFlexGap>
              <Chip size="small" label="hub" sx={{ borderColor: '#5eead4', color: '#5eead4' }} variant="outlined" />
              <Chip size="small" label="device online" sx={{ borderColor: '#38bdf8', color: '#38bdf8' }} variant="outlined" />
              <Chip size="small" label="device offline" sx={{ borderColor: '#64748b', color: '#64748b' }} variant="outlined" />
            </Stack>
          </Paper>

          <TableContainer component={Paper} sx={{ background: '#0f172a' }}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Type</TableCell>
                  <TableCell>Hostname</TableCell>
                  <TableCell>Address</TableCell>
                  <TableCell>Ranges</TableCell>
                  <TableCell>OS</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Last seen</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {graph.nodes.map((n) => (
                  <TableRow key={n.id} hover sx={{ '& td': { borderColor: 'rgba(148,163,184,0.12)' } }}>
                    <TableCell>
                      <Chip
                        size="small"
                        label={n.type}
                        sx={{
                          textTransform: 'uppercase',
                          fontSize: 10,
                          bgcolor: n.type === 'hub' ? 'rgba(94,234,212,0.12)' : 'rgba(56,189,248,0.12)',
                          color: n.type === 'hub' ? '#5eead4' : '#38bdf8',
                        }}
                      />
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'IBM Plex Mono, monospace', fontWeight: 600 }}>
                      {n.hostname || n.label}
                    </TableCell>
                    <TableCell><code>{n.address}</code></TableCell>
                    <TableCell sx={{ maxWidth: 280 }}>
                      <Typography variant="caption" color="text.secondary">
                        {(n.cidrs || []).join(', ')}
                      </Typography>
                    </TableCell>
                    <TableCell>{n.os || '—'}</TableCell>
                    <TableCell>{statusChip(n)}</TableCell>
                    <TableCell>
                      {n.lastSeen ? new Date(n.lastSeen).toLocaleString() : '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}
    </Box>
  );
};

export default Topology;
