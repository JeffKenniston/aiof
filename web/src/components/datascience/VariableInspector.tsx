export default function VariableInspector({ onClose }: { onClose: () => void }) {
  return (
    <div className="w-[320px] h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-4 border-b border-white/5 bg-white/[0.01]">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Environment</span>
        <button onClick={onClose} className="text-text-muted hover:text-white"><svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg></button>
      </div>
      
      <div className="p-4 space-y-6">
        <div>
          <h4 className="text-xs font-medium text-text-muted mb-3 uppercase tracking-widest">DataFrames</h4>
          <div className="p-3 rounded-lg bg-white/5 border border-white/5 hover:border-emerald-500/30 transition-colors cursor-pointer group">
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm text-emerald-400 font-mono">df</span>
              <span className="text-[10px] bg-white/10 px-1.5 py-0.5 rounded text-text-muted">pandas.DataFrame</span>
            </div>
            <div className="text-xs text-text-secondary font-mono">Shape: (12000, 4)</div>
            <div className="mt-3 pt-3 border-t border-white/5 text-[10px] text-text-muted grid grid-cols-2 gap-1 font-mono">
              <span className="text-emerald-400/70">Date:</span> <span>datetime64</span>
              <span className="text-emerald-400/70">Ticker:</span> <span>object</span>
              <span className="text-emerald-400/70">Revenue:</span> <span>float64</span>
              <span className="text-emerald-400/70">EPS:</span> <span>float64</span>
            </div>
          </div>
        </div>
        
        <div>
          <h4 className="text-xs font-medium text-text-muted mb-3 uppercase tracking-widest">Memory State</h4>
          <div className="space-y-4">
            <div className="space-y-1 text-xs">
              <div className="flex justify-between">
                <span className="text-text-secondary">RAM Usage</span>
                <span className="text-emerald-400 font-mono">458 MB</span>
              </div>
              <div className="w-full h-1.5 bg-white/5 rounded-full overflow-hidden">
                <div className="h-full bg-emerald-500/50 w-1/3" />
              </div>
            </div>
            
            <div className="space-y-1 text-xs">
              <div className="flex justify-between">
                <span className="text-text-secondary">Context Tokens</span>
                <span className="text-accent-blue font-mono">14,230</span>
              </div>
              <div className="w-full h-1.5 bg-white/5 rounded-full overflow-hidden">
                <div className="h-full bg-accent-blue/50 w-[10%]" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
