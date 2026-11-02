import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert, Box, CircularProgress, Grid, Paper, ToggleButton, ToggleButtonGroup, Typography,
} from '@mui/material';
import { ResponsiveLine } from '@nivo/line';
import { api } from '../api';

interface Stats {
  vscales: number;
  hubs: number;
  hubsOnline: number;
  devices: number;
  devicesOnline: number;
  routes: number;
  transferRx: number;
  transferTx: number;
}

interface Point {
  takenAt: string;
  vscales: number;
  hubs: number;
  hubsOnline: number;
  devices: number;
  devicesOnline: number;
  routes: number;
  transferRx: number;
  transferTx: number;
  transferRxRate: number;
  transferTxRate: number;
}

const RANGES = ['1h', '7h', '12h', '24h', '7d'] as const;
type Range = typeof RANGES[number];

const Card: React.FC<{ title: string; value: string | number; sub?: string; accent?: string }> = ({
  title, value, sub, accent = '#5eead4',
}) => (
  <Paper sx={{
    p: 2.5, height: '100%',
    background: 'linear-gradient(145deg,#111827,#0f172a)',
    borderTop: `2px solid ${accent}`,
  }}>
    <Typography variant="body2" color="text.secondary">{title}</Typography>
    <Typography variant="h4" sx={{ mt: 1, fontFamily: 'IBM Plex Mono, monospace' }}>{value}</Typography>
    {sub && <Typography variant="caption" color="text.secondary">{sub}</Typography>}
  </Paper>
);

function formatBytes(n: number) {
  if (!n) return '0 B';
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), u.length - 1);
  return `${(n / Math.pow(1024, i)).toFixed(1)} ${u[i]}`;
}

function formatRate(n: number) {
  return `${formatBytes(n)}/s`;
}

const chartTheme = {
  text: { fill: '#94a3b8', fontSize: 11 },
  axis: {
    ticks: { text: { fill: '#94a3b8' } },
    legend: { text: { fill: '#cbd5e1' } },
  },
  grid: { line: { stroke: 'rgba(148,163,184,0.12)' } },
  crosshair: { line: { stroke: '#5eead4' } },
  tooltip: {
    container: { background: '#0f172a', color: '#e2e8f0', border: '1px solid #334155' },
  },
};

const ChartCard: React.FC<{ title: string; children: React.ReactNode }> = ({ title, children }) => (
  <Paper sx={{ p: 2, height: 320, background: '#0f172a' }}>
    <Typography variant="subtitle2" color="text.secondary" mb={1}>{title}</Typography>
    <Box sx={{ height: 260 }}>{children}</Box>
  </Paper>
);

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<Stats | null>(null);
  const [points, setPoints] = useState<Point[]>([]);
  const [range, setRange] = useState<Range>('24h');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      api.get('/api/stats'),
      api.get(`/api/stats/history?range=${range}`),
    ])
      .then(([s, h]) => {
        setStats(s.data);
        setPoints(h.data.points || []);
      })
      .catch(() => setError('Failed to load dashboard'))
      .finally(() => setLoading(false));
  }, [range]);

  const onlineSeries = useMemo(() => [
    {
      id: 'devices online',
      color: '#38bdf8',
      data: points.map((p) => ({ x: p.takenAt, y: p.devicesOnline })),
    },
    {
      id: 'hubs online',
      color: '#5eead4',
      data: points.map((p) => ({ x: p.takenAt, y: p.hubsOnline })),
    },
  ], [points]);

  const inventorySeries = useMemo(() => [
    {
      id: 'devices',
      color: '#a78bfa',
      data: points.map((p) => ({ x: p.takenAt, y: p.devices })),
    },
    {
      id: 'hubs',
      color: '#fbbf24',
      data: points.map((p) => ({ x: p.takenAt, y: p.hubs })),
    },
    {
      id: 'vscales',
      color: '#f472b6',
      data: points.map((p) => ({ x: p.takenAt, y: p.vscales })),
    },
  ], [points]);

  const rateSeries = useMemo(() => [
    {
      id: 'rx rate',
      color: '#34d399',
      data: points.map((p) => ({ x: p.takenAt, y: p.transferRxRate })),
    },
    {
      id: 'tx rate',
      color: '#fb7185',
      data: points.map((p) => ({ x: p.takenAt, y: p.transferTxRate })),
    },
  ], [points]);

  if (loading && !stats) {
    return <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress /></Box>;
  }
  if (error) return <Alert severity="error">{error}</Alert>;
  if (!stats) return null;

  const lineProps = {
    margin: { top: 10, right: 20, bottom: 40, left: 50 },
    xScale: { type: 'time' as const, format: 'native', precision: 'minute' as const },
    xFormat: 'time:%H:%M',
    yScale: { type: 'linear' as const, min: 0, stacked: false },
    axisBottom: {
      format: range === '7d' ? '%m/%d' : '%H:%M',
      tickRotation: -30,
      legendOffset: 36,
    },
    axisLeft: { tickSize: 0 },
    enablePoints: false,
    enableGridX: false,
    useMesh: true,
    theme: chartTheme,
    colors: { datum: 'color' as const },
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, flexWrap: 'wrap', gap: 1 }}>
        <Typography variant="h5">Dashboard</Typography>
        <ToggleButtonGroup
          exclusive
          size="small"
          value={range}
          onChange={(_, v) => v && setRange(v)}
          sx={{
            '& .MuiToggleButton-root': { color: '#94a3b8', borderColor: '#334155' },
            '& .Mui-selected': { color: '#5eead4 !important', bgcolor: 'rgba(94,234,212,0.12) !important' },
          }}
        >
          {RANGES.map((r) => <ToggleButton key={r} value={r}>{r}</ToggleButton>)}
        </ToggleButtonGroup>
      </Box>

      <Grid container spacing={2} mb={2}>
        <Grid item xs={12} sm={6} md={3}><Card title="Vscales" value={stats.vscales} accent="#f472b6" /></Grid>
        <Grid item xs={12} sm={6} md={3}><Card title="Hubs online" value={`${stats.hubsOnline}/${stats.hubs}`} accent="#5eead4" /></Grid>
        <Grid item xs={12} sm={6} md={3}><Card title="Devices online" value={`${stats.devicesOnline}/${stats.devices}`} accent="#38bdf8" /></Grid>
        <Grid item xs={12} sm={6} md={3}><Card title="Routes" value={stats.routes} accent="#fbbf24" /></Grid>
        <Grid item xs={12} sm={6} md={3}><Card title="Transfer RX" value={formatBytes(stats.transferRx)} accent="#34d399" /></Grid>
        <Grid item xs={12} sm={6} md={3}><Card title="Transfer TX" value={formatBytes(stats.transferTx)} accent="#fb7185" /></Grid>
      </Grid>

      {points.length === 0 ? (
        <Alert severity="info">Collecting metrics… charts populate after the first minute of samples.</Alert>
      ) : (
        <Grid container spacing={2}>
          <Grid item xs={12} md={6}>
            <ChartCard title={`Online nodes (${range})`}>
              <ResponsiveLine
                {...lineProps}
                data={onlineSeries.map((s) => ({
                  ...s,
                  data: s.data.map((d) => ({ x: new Date(d.x), y: d.y })),
                }))}
                yFormat=" >-.0f"
              />
            </ChartCard>
          </Grid>
          <Grid item xs={12} md={6}>
            <ChartCard title={`Inventory (${range})`}>
              <ResponsiveLine
                {...lineProps}
                data={inventorySeries.map((s) => ({
                  ...s,
                  data: s.data.map((d) => ({ x: new Date(d.x), y: d.y })),
                }))}
                yFormat=" >-.0f"
              />
            </ChartCard>
          </Grid>
          <Grid item xs={12}>
            <ChartCard title={`Throughput rate (${range})`}>
              <ResponsiveLine
                {...lineProps}
                data={rateSeries.map((s) => ({
                  ...s,
                  data: s.data.map((d) => ({ x: new Date(d.x), y: d.y })),
                }))}
                yFormat={(v) => formatRate(Number(v))}
              />
            </ChartCard>
          </Grid>
        </Grid>
      )}
    </Box>
  );
};

export default Dashboard;
