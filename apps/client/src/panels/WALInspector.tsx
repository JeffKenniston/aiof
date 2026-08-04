import { useState, Fragment, use, Suspense, useMemo } from 'react';
import { getDatabase, type WALEventDocType } from '../db/index';
import { useRxQuery } from '../hooks/useRxQuery';

function WALInspectorContent({ }: { panelId: string }) {
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  // React 19 Suspense: Unwrapping the DB promise directly using `use`
  const db = use(getDatabase()); 

  const query = useMemo(() => db.wal_events.find({
    sort: [{ updatedAt: 'desc' }],
    limit: 100
  }), [db]);

  const events = useRxQuery<WALEventDocType[]>(query as any) || [];

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'INSERT': return 'text-accent-green bg-accent-green/10 border-accent-green/20';
      case 'UPDATE': return 'text-accent-blue bg-accent-blue/10 border-accent-blue/20';
      case 'DELETE': return 'text-accent-red bg-accent-red/10 border-accent-red/20';
      default: return 'text-text-muted bg-surface-overlay border-border-default';
    }
  };

  return (
    <div className="w-full h-full bg-surface-base flex flex-col font-mono text-xs">
      {/* Toolbar */}
      <div className="h-10 bg-surface-raised border-b border-border-default flex items-center px-4 shrink-0 gap-4">
        <button className="flex items-center gap-1.5 text-text-muted hover:text-text-primary transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
          Refresh
        </button>
        <div className="w-px h-4 bg-border-default"></div>
        <select className="bg-transparent text-text-primary focus:outline-none cursor-pointer">
          <option>Last 15 minutes</option>
          <option>Last 1 hour</option>
          <option>All time</option>
        </select>
        <span className="ml-auto text-text-muted">{events.length} mutations</span>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-left border-collapse">
          <thead className="bg-surface-overlay sticky top-0 border-b border-border-default z-10">
            <tr>
              <th className="p-2 font-semibold text-text-secondary w-20">Type</th>
              <th className="p-2 font-semibold text-text-secondary">ID</th>
              <th className="p-2 font-semibold text-text-secondary">Updated At</th>
              <th className="p-2 font-semibold text-text-secondary w-16 text-center">Del</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border-subtle">
            {events.map((ev) => (
              <Fragment key={ev.id}>
                <tr 
                  className="hover:bg-surface-raised cursor-pointer transition-colors group"
                  onClick={() => setExpandedRow(expandedRow === ev.id ? null : ev.id)}
                >
                  <td className="p-2">
                    <span className={`px-2 py-0.5 rounded border ${getTypeColor(ev.type)}`}>
                      {ev.type}
                    </span>
                  </td>
                  <td className="p-2 text-text-primary truncate max-w-[200px]" title={ev.id}>{ev.id}</td>
                  <td className="p-2 text-text-muted">
                    {new Date(ev.updatedAt).toLocaleTimeString()}
                  </td>
                  <td className="p-2 text-center">
                    {ev.deleted ? (
                      <span className="text-accent-red">✓</span>
                    ) : (
                      <span className="text-text-muted">-</span>
                    )}
                  </td>
                </tr>
                {expandedRow === ev.id && (
                  <tr>
                    <td colSpan={4} className="p-0 border-b border-border-default">
                      <div className="bg-surface-glass p-4 border-l-2 border-accent-blue m-2 rounded">
                        <pre className="text-text-primary overflow-x-auto">
                          {JSON.stringify(ev.payload, null, 2)}
                        </pre>
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
            {events.length === 0 && (
              <tr>
                <td colSpan={4} className="p-8 text-center text-text-muted">
                  No WAL events recorded.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default function WALInspector({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted font-mono text-xs">Loading WAL...</div>}>
      <WALInspectorContent panelId={panelId} />
    </Suspense>
  );
}
