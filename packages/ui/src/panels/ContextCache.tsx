import { getDatabase, useRxQuery } from '@aiof/rxdb-store';
import { useState, useEffect, use, Suspense, useMemo } from 'react';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';

function ContextCacheContent({ }: { panelId: string }) {
  const [now, setNow] = useState(Date.now() / 1000);

  const db = use(getDatabase());
  const query = useMemo(() => db.context_cache.find(), [db]);
  const caches = useRxQuery<ContextCacheDocType[]>(query as any) || [];

  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now() / 1000), 1000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="w-full h-full bg-surface-base p-4 overflow-y-auto">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-lg font-semibold text-text-primary flex items-center gap-2">
          <svg className="w-5 h-5 text-accent-purple" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" /></svg>
          Active Context Caches
        </h2>
        <button className="px-3 py-1.5 bg-accent-blue/10 text-accent-blue border border-accent-blue/30 hover:bg-accent-blue/20 rounded text-sm font-medium transition-colors flex items-center gap-1">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" /></svg>
          Create Cache
        </button>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        {caches.map(cache => {
          const ttlRemaining = Math.max(0, Math.floor((cache.createdAt + cache.ttlSeconds) - now));
          const pctUsed = Math.min(100, (cache.tokenCount / cache.contextWindow) * 100);
          
          return (
            <div key={cache.id} className="bg-surface-raised border border-border-default rounded-xl p-4 shadow-panel group hover:border-accent-purple/50 transition-colors">
              <div className="flex justify-between items-start mb-3">
                <div>
                  <h3 className="font-semibold text-text-primary text-base">{cache.name}</h3>
                  <span className="text-xs text-text-muted bg-surface-overlay px-2 py-0.5 rounded border border-border-subtle inline-block mt-1">
                    {cache.model}
                  </span>
                </div>
                <div className="text-right">
                  <div className={`text-xl font-bold font-mono ${ttlRemaining < 300 ? 'text-accent-red' : 'text-accent-blue'}`}>
                    {Math.floor(ttlRemaining / 60)}:{String(ttlRemaining % 60).padStart(2, '0')}
                  </div>
                  <div className="text-xs text-text-muted">TTL Remaining</div>
                </div>
              </div>

              <div className="space-y-2 mb-4">
                <div className="flex justify-between text-xs text-text-secondary">
                  <span>Tokens ({cache.tokenCount.toLocaleString()} / {cache.contextWindow.toLocaleString()})</span>
                  <span>{pctUsed.toFixed(1)}%</span>
                </div>
                <div className="w-full h-2 bg-surface-overlay rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-accent-purple rounded-full"
                    style={{ width: `${pctUsed}%` }}
                  />
                </div>
              </div>

              <div className="flex items-center gap-2 text-xs text-text-muted bg-surface-overlay p-2 rounded border border-border-subtle">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" /></svg>
                {cache.agentCount} Agents connected
              </div>
            </div>
          );
        })}
        {caches.length === 0 && (
          <div className="col-span-full flex flex-col items-center justify-center h-32 text-text-muted border border-dashed border-border-default rounded-xl bg-surface-raised/50">
            No active context caches.
          </div>
        )}
      </div>
    </div>
  );
}

export default function ContextCache({ panelId }: { panelId: string }) {
  return (
    <Suspense fallback={<div className="p-4 text-text-muted">Loading Context Caches...</div>}>
      <ContextCacheContent panelId={panelId} />
    </Suspense>
  );
}
