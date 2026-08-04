import { getDatabase, useRxQuery, useLayoutStore } from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';
import { use } from 'react';

export default function KPIWidget(props: { label: string, collectionName: 'telemetry' | 'task_queue' | 'agent_logs', icon: 'mail' | 'clock' }) {
  const activeProject = useLayoutStore((state) => state.activeProject);
  
  if (!activeProject) {
    return <div className="bg-surface-base/40 backdrop-blur-3xl border border-white/5 p-5 rounded-3xl shadow-lg relative overflow-hidden group">
      <div className="text-text-muted text-xs">No project selected</div>
    </div>;
  }

  return <KPIWidgetContent {...props} />;
}

function KPIWidgetContent({ label, collectionName, icon }: { label: string, collectionName: 'telemetry' | 'task_queue' | 'agent_logs', icon: 'mail' | 'clock' }) {
  const db = use(getDatabase());
  
  // Real implementation fetching from RxDB
  const query = db[collectionName]?.find().limit(100);
  const data = useRxQuery<any[]>(query as any) || [];

  const value = data ? data.length.toString() : '...';
  const trend = '+0%'; 

  return (
    <div className="bg-surface-base/40 backdrop-blur-3xl border border-white/5 p-5 rounded-3xl shadow-lg relative overflow-hidden group hover:border-amber-500/30 transition-colors">
      <div className="absolute -right-4 -top-4 w-24 h-24 bg-white/5 rounded-full blur-2xl group-hover:bg-amber-500/10 transition-colors" />
      <div className="text-text-muted mb-2">
        {icon === 'mail' && <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg>}
        {icon === 'clock' && <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>}
      </div>
      <div className="text-3xl font-light text-text-primary mb-1">{value}</div>
      <div className="flex items-center justify-between mt-3">
        <div className="text-[10px] uppercase tracking-wider text-text-muted font-medium">{label}</div>
        <div className="text-[10px] text-emerald-400 font-bold bg-emerald-500/10 px-2 py-0.5 rounded">{trend}</div>
      </div>
    </div>
  );
}
