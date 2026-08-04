import { getDatabase, useRxQuery } from '@aiof/rxdb-store';
import { useState, useRef, useEffect, use, Suspense, useMemo } from 'react';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';

function AgentTextLogContent({ }: { panelId: string }) {
  const [filter, setFilter] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  const db = use(getDatabase());
  const query = useMemo(() => db.agent_logs.find({ sort: [{ timestamp: 'asc' }] }), [db]);
  const logs = useRxQuery<AgentLogDocType[]>(query as any) || [];

  useEffect(() => {
    setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 100);
  }, [logs]);

  const getSeverityColors = (sev: string) => {
    switch (sev) {
      case 'info': return 'text-accent-blue bg-accent-blue/10 border-accent-blue/20';
      case 'warn': return 'text-accent-amber bg-accent-amber/10 border-accent-amber/20';
      case 'error': return 'text-accent-red bg-accent-red/10 border-accent-red/20';
      default: return 'text-text-muted bg-surface-overlay border-border-default';
    }
  };

  const filteredLogs = logs.filter(l => 
    l.message.toLowerCase().includes(filter.toLowerCase()) || 
    l.agent.toLowerCase().includes(filter.toLowerCase())
  );

  return (
    <div className="w-full h-full flex flex-col bg-surface-base font-mono text-xs">
      <div className="p-2 border-b border-border-default bg-surface-raised flex items-center gap-2 shrink-0">
        <input 
          type="text"
          placeholder="Filter logs..."
          className="bg-surface-base border border-border-default text-text-primary px-2 py-1 rounded w-full max-w-xs focus:outline-none focus:border-accent-blue transition-colors"
          value={filter}
          onChange={e => setFilter(e.target.value)}
        />
        <span className="text-text-muted ml-auto">{filteredLogs.length} events</span>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {filteredLogs.map((log) => (
          <div key={log.id} className="flex gap-3 hover:bg-surface-raised p-1 rounded transition-colors group">
            <div className="text-text-muted shrink-0 w-20">
              {new Date(log.timestamp).toLocaleTimeString([], { hour12: false, fractionalSecondDigits: 2 })}
            </div>
            <div className={`shrink-0 w-12 text-center rounded border ${getSeverityColors(log.severity)}`}>
              {log.severity.toUpperCase()}
            </div>
            <div className="shrink-0 w-24 text-text-secondary font-semibold truncate" title={log.agent}>
              [{log.agent}]
            </div>
            <div className="flex-1 text-text-primary break-all">
              {log.message}
              {log.reasoning && log.reasoning.length > 0 && (
                <div className="mt-1 pl-2 border-l-2 border-border-default text-text-muted">
                  {log.reasoning.map((r, i) => <div key={i}>&gt; {r}</div>)}
                </div>
              )}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

export default function AgentTextLog({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted font-mono text-xs">Loading Logs...</div>}>
      <AgentTextLogContent panelId={panelId} />
    </Suspense>
  );
}
