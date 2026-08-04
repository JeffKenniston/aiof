import { getDatabase, useLayoutStore } from '@aiof/rxdb-store';
import { useState, useEffect } from 'react';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';

export default function DiagnosticCard() {
  const activeProject = useLayoutStore((s) => s.activeProject);
  const [isHydrated, setIsHydrated] = useState(false);
  const [mutationsCount, setMutationsCount] = useState(0);

  useEffect(() => {
    if (!activeProject) {
      setIsHydrated(false);
      return;
    }
    let active = true;
    getDatabase(activeProject).then(() => {
      if (active) {
        setIsHydrated(true);
      }
    }).catch(err => {
      console.error('Failed to initialize db in DiagnosticCard:', err);
    });
    return () => {
      active = false;
    };
  }, [activeProject]);

  const handleDispatchMutation = async () => {
    try {
      const db = await getDatabase(activeProject);
      const timestamp = Date.now();
      await db.wal_events.insert({
        id: `mutation_${timestamp}_${Math.random().toString(36).substring(2, 7)}`,
        type: 'INSERT',
        updatedAt: timestamp,
        deleted: false,
        payload: {
          timestamp,
          source: 'ui_chaos_test',
          state: 'mutated',
          description: 'Chaos engineering local write assertion'
        }
      });
      setMutationsCount(prev => prev + 1);
      console.log('Local state mutation written successfully to RxDB.');
    } catch (err) {
      console.error('Failed to dispatch local mutation to RxDB:', err);
    }
  };

  if (!isHydrated) return null;

  return (
    <div className="absolute bottom-16 right-4 z-50 flex flex-col gap-2 p-3 bg-surface-glass border border-border-default rounded-lg text-xs backdrop-blur-md shadow-elevated animate-in fade-in slide-in-from-bottom-2 duration-300 pointer-events-auto">
      <div className="flex items-center gap-2">
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-accent-green opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-accent-green"></span>
        </span>
        <span className="font-mono text-text-secondary">
          Status: WAL Hydrated & RxDB Initialized
        </span>
      </div>
      <div className="flex items-center justify-between gap-4 mt-1 border-t border-border-subtle pt-2">
        <span className="text-[10px] text-text-muted font-mono">
          Mutations: {mutationsCount}
        </span>
        <button
          onClick={handleDispatchMutation}
          className="px-2.5 py-1 bg-accent-blue/10 hover:bg-accent-blue/20 text-accent-blue border border-accent-blue/30 rounded text-[10px] font-semibold transition-colors duration-200 cursor-pointer"
        >
          Dispatch Local State Mutation
        </button>
      </div>
    </div>
  );
}
