import { use, Suspense, useMemo } from 'react';
import { getDatabase, type SandboxStateDocType } from '../db/index';
import { useRxQuery } from '../hooks/useRxQuery';

function SandboxMonitorContent({ }: { panelId: string }) {
  const db = use(getDatabase());
  const query = useMemo(() => db.sandbox_state.find(), [db]);
  const containers = useRxQuery<SandboxStateDocType[]>(query as any) || [];

  const formatUptime = (seconds: number) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return `${h}h ${m}m ${s}s`;
  };

  return (
    <div className="w-full h-full bg-surface-base p-4 overflow-y-auto">
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {containers.map(container => (
          <div key={container.id} className="bg-surface-raised border border-border-default rounded-xl p-4 flex flex-col gap-4 shadow-panel">
            
            {/* Header */}
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold text-text-primary text-base">{container.name}</h3>
                <p className="text-xs text-text-muted font-mono">{container.id}</p>
              </div>
              <div className={`px-2 py-1 rounded text-xs font-bold uppercase border ${
                container.status === 'running' ? 'text-accent-green bg-accent-green/10 border-accent-green/20' :
                container.status === 'error' ? 'text-accent-red bg-accent-red/10 border-accent-red/20' :
                'text-text-muted bg-surface-overlay border-border-default'
              }`}>
                {container.status}
              </div>
            </div>

            {/* Metrics */}
            <div className="space-y-3">
              <div>
                <div className="flex justify-between text-xs text-text-secondary mb-1">
                  <span>CPU Usage</span>
                  <span>{container.cpuPercent.toFixed(1)}%</span>
                </div>
                <div className="w-full h-1.5 bg-surface-overlay rounded-full overflow-hidden">
                  <div 
                    className={`h-full rounded-full transition-all duration-500 ${container.cpuPercent > 80 ? 'bg-accent-red' : 'bg-accent-blue'}`}
                    style={{ width: `${Math.min(container.cpuPercent, 100)}%` }}
                  />
                </div>
              </div>
              <div>
                <div className="flex justify-between text-xs text-text-secondary mb-1">
                  <span>Memory ({container.memoryMb}MB / {container.memoryLimitMb}MB)</span>
                  <span>{container.memoryPercent.toFixed(1)}%</span>
                </div>
                <div className="w-full h-1.5 bg-surface-overlay rounded-full overflow-hidden">
                  <div 
                    className={`h-full rounded-full transition-all duration-500 ${container.memoryPercent > 80 ? 'bg-accent-red' : 'bg-accent-purple'}`}
                    style={{ width: `${Math.min(container.memoryPercent, 100)}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Uptime & Controls */}
            <div className="flex justify-between items-center text-xs pt-2 border-t border-border-subtle">
              <span className="text-text-muted font-mono">Up: {formatUptime(container.uptimeSeconds)}</span>
              <div className="flex gap-2">
                <button className="w-7 h-7 flex items-center justify-center rounded bg-surface-overlay hover:bg-surface-raised text-text-secondary hover:text-accent-blue border border-border-default transition-colors" title="Restart">
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                </button>
                <button className="w-7 h-7 flex items-center justify-center rounded bg-surface-overlay hover:bg-surface-raised text-text-secondary hover:text-accent-red border border-border-default transition-colors" title="Stop">
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" /></svg>
                </button>
              </div>
            </div>

          </div>
        ))}
        {containers.length === 0 && (
          <div className="col-span-full flex flex-col items-center justify-center h-48 text-text-muted">
            <svg className="w-12 h-12 mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" /></svg>
            <p>No active sandbox containers.</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default function SandboxMonitor({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted">Loading Sandbox metrics...</div>}>
      <SandboxMonitorContent panelId={panelId} />
    </Suspense>
  );
}
