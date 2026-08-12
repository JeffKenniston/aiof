export default function DatasetSidebar({ onClose }: { onClose: () => void }) {
  return (
    <div className="w-[280px] h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-4 border-b border-white/5">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Data Sources</span>
        <button onClick={onClose} className="text-text-muted hover:text-white"><svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg></button>
      </div>
      <div className="p-4 space-y-4">
        <button className="w-full py-2 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-sm font-medium rounded-lg transition-colors border border-emerald-500/20 flex justify-center items-center gap-2">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" /></svg>
          Add Source
        </button>
        
        {/* Real Sources */}
        <div className="space-y-2">
          {/* This would normally map over a state array, e.g. sources.map(...) */}
          <div className="flex flex-col items-center justify-center py-8 text-center px-4">
            <svg className="w-8 h-8 text-white/20 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
            </svg>
            <p className="text-sm text-text-muted">No data sources added yet.</p>
            <p className="text-xs text-white/30 mt-1">Click above to connect a source.</p>
          </div>
        </div>
      </div>
    </div>
  );
}
