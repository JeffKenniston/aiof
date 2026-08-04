import { getDatabase, type TelemetryDocType, useRxQuery } from '@aiof/rxdb-store';
import { use, Suspense, useMemo } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line } from 'recharts';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';

function TelemetryContent({ }: { panelId: string }) {
  const db = use(getDatabase());
  const query = useMemo(() => db.telemetry.find({
    sort: [{ timestamp: 'asc' }],
    limit: 60
  }), [db]);
  const data = useRxQuery<TelemetryDocType[]>(query as any) || [];

  const agentsQuery = useMemo(() => db.agent_graph_state.find(), [db]);
  const agentsData = useRxQuery<AgentLogDocType[]>(agentsQuery as any) || [];
  const activeAgentCount = agentsData.length > 0 ? agentsData.length : 15; // fallback

  const latest = data[data.length - 1] || { reqPerSec: 0, errorPct: 0, p50: 0 };

  return (
    <div className="w-full h-full bg-surface-base p-4 flex flex-col gap-4 overflow-y-auto">
      
      {/* Stat Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 shrink-0">
        <div className="bg-surface-raised border border-border-default rounded-lg p-3">
          <div className="text-text-muted text-xs mb-1">Req / Sec</div>
          <div className="text-2xl font-bold text-text-primary">{latest.reqPerSec?.toFixed(1) || '0.0'}</div>
        </div>
        <div className="bg-surface-raised border border-border-default rounded-lg p-3">
          <div className="text-text-muted text-xs mb-1">Error Rate</div>
          <div className={`text-2xl font-bold ${(latest.errorPct || 0) > 5 ? 'text-accent-red' : 'text-accent-green'}`}>
            {latest.errorPct?.toFixed(2) || '0.00'}%
          </div>
        </div>
        <div className="bg-surface-raised border border-border-default rounded-lg p-3">
          <div className="text-text-muted text-xs mb-1">P50 Latency</div>
          <div className="text-2xl font-bold text-accent-blue">{latest.p50 || 0}ms</div>
        </div>
        <div className="bg-surface-raised border border-border-default rounded-lg p-3">
          <div className="text-text-muted text-xs mb-1">Active Agents</div>
          <div className="text-2xl font-bold text-accent-purple">{activeAgentCount}</div>
        </div>
      </div>

      {/* Charts */}
      <div className="flex-1 min-h-[200px] bg-surface-raised border border-border-default rounded-lg p-4 flex flex-col">
        <h3 className="text-sm font-semibold text-text-secondary mb-4">Throughput (Req/sec)</h3>
        <div className="flex-1">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data}>
              <defs>
                <linearGradient id="colorReq" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                  <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
              <XAxis dataKey="time" stroke="#64748b" fontSize={10} tickMargin={8} />
              <YAxis stroke="#64748b" fontSize={10} tickFormatter={val => `${val}`} />
              <Tooltip 
                contentStyle={{ backgroundColor: '#111827', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px' }}
                itemStyle={{ color: '#f1f5f9' }}
              />
              <Area type="monotone" dataKey="reqPerSec" stroke="#3b82f6" strokeWidth={2} fillOpacity={1} fill="url(#colorReq)" isAnimationActive={false} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="flex-1 min-h-[200px] bg-surface-raised border border-border-default rounded-lg p-4 flex flex-col">
        <h3 className="text-sm font-semibold text-text-secondary mb-4">Latency (P50, P95, P99)</h3>
        <div className="flex-1">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
              <XAxis dataKey="time" stroke="#64748b" fontSize={10} tickMargin={8} />
              <YAxis stroke="#64748b" fontSize={10} />
              <Tooltip 
                contentStyle={{ backgroundColor: '#111827', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px' }}
              />
              <Line type="monotone" dataKey="p50" stroke="#3b82f6" strokeWidth={2} dot={false} isAnimationActive={false} />
              <Line type="monotone" dataKey="p95" stroke="#a855f7" strokeWidth={2} dot={false} isAnimationActive={false} />
              <Line type="monotone" dataKey="p99" stroke="#ef4444" strokeWidth={2} dot={false} isAnimationActive={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

    </div>
  );
}

export default function Telemetry({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted">Loading Telemetry...</div>}>
      <TelemetryContent panelId={panelId} />
    </Suspense>
  );
}
