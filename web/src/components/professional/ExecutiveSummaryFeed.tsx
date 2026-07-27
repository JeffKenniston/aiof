import { useRxQuery } from '../../hooks/useRxQuery';
import { getDatabase } from '../../db';
import { use } from 'react';
import { useLayoutStore } from '../../hooks/useLayoutStore';

export default function ExecutiveSummaryFeed() {
  const activeProject = useLayoutStore((state) => state.activeProject);
  
  if (!activeProject) {
    return <div className="p-4 text-text-muted">No project selected</div>;
  }

  return <ExecutiveSummaryFeedContent />;
}

function ExecutiveSummaryFeedContent() {
  const db = use(getDatabase());
  const logsQuery = db.agent_logs.find().sort({ timestamp: 'desc' }).limit(5);
  const logs = useRxQuery<any[]>(logsQuery as any) || [];

  if (!logs) return <div className="p-4 text-text-muted">Loading logs...</div>;

  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {logs.length === 0 ? (
        <div className="text-sm text-text-muted text-center p-4">No recent activity</div>
      ) : logs.map((log: any) => (
        <div key={log.id} className="p-4 rounded-2xl bg-white/5 border border-white/5 hover:bg-white/10 transition-colors cursor-pointer group">
          <div className="flex items-center gap-2 mb-2">
            <span className={`px-2 py-0.5 rounded text-[9px] uppercase tracking-wider font-semibold ${log.severity === 'error' ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'}`}>
              {log.agent || 'System'}
            </span>
            <span className="text-[10px] text-text-muted">{new Date(log.timestamp).toLocaleTimeString()}</span>
          </div>
          <h4 className="text-sm font-medium text-text-primary mb-1 group-hover:text-amber-500 transition-colors">{log.message}</h4>
        </div>
      ))}
    </div>
  );
}
