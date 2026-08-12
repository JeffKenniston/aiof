export default function AssetLibrary({ onClose }: { onClose: () => void }) {
  return (
    <div className="w-[280px] h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-4 border-b border-white/5 bg-white/[0.01]">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Gallery</span>
        <button onClick={onClose} className="text-text-muted hover:text-white transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      
      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        <div>
          <h4 className="text-[10px] uppercase tracking-widest text-text-muted mb-3 font-semibold flex items-center gap-2">
            <svg className="w-3 h-3 text-fuchsia-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
            Today
          </h4>
          <div className="grid grid-cols-2 gap-3">
            <div className="aspect-square rounded-xl bg-surface-raised border border-white/5 overflow-hidden relative group cursor-pointer hover:border-fuchsia-500/50 transition-colors shadow-sm">
              <div className="absolute inset-0 bg-gradient-to-br from-fuchsia-500/20 to-violet-500/20" />
              <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity bg-black/40 flex items-center justify-center backdrop-blur-[2px]">
                <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
              </div>
            </div>
            <div className="aspect-square rounded-xl bg-surface-raised border border-white/5 overflow-hidden relative group cursor-pointer hover:border-fuchsia-500/50 transition-colors shadow-sm">
              <div className="absolute inset-0 bg-gradient-to-br from-fuchsia-500/10 to-violet-500/10" />
            </div>
            <div className="aspect-square rounded-xl bg-surface-raised border border-white/5 overflow-hidden relative group cursor-pointer hover:border-fuchsia-500/50 transition-colors shadow-sm">
              <div className="absolute inset-0 bg-gradient-to-br from-fuchsia-500/15 to-violet-500/5" />
            </div>
          </div>
        </div>

        <div>
          <h4 className="text-[10px] uppercase tracking-widest text-text-muted mb-3 font-semibold flex items-center gap-2">
             <svg className="w-3 h-3 text-text-secondary" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            Yesterday
          </h4>
          <div className="grid grid-cols-2 gap-3 opacity-60">
            <div className="aspect-square rounded-xl bg-surface-raised border border-white/5 overflow-hidden relative group cursor-pointer hover:border-fuchsia-500/50 transition-colors shadow-sm">
              <div className="absolute inset-0 bg-gradient-to-br from-blue-500/10 to-purple-500/10" />
            </div>
            <div className="aspect-square rounded-xl bg-surface-raised border border-white/5 overflow-hidden relative group cursor-pointer hover:border-fuchsia-500/50 transition-colors shadow-sm">
              <div className="absolute inset-0 bg-gradient-to-br from-emerald-500/10 to-teal-500/10" />
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
