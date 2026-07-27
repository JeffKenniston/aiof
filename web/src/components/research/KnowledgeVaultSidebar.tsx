export default function KnowledgeVaultSidebar({ onClose }: { onClose: () => void }) {
  return (
    <div className="w-[300px] h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-4 border-b border-white/5 bg-white/[0.01]">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Knowledge Vault</span>
        <button onClick={onClose} className="text-text-muted hover:text-white transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        
        {/* NotebookLM Style Audio Generator */}
        <div className="p-4 rounded-2xl bg-gradient-to-br from-blue-500/10 to-purple-500/10 border border-blue-500/20 relative overflow-hidden group">
          <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/10 rounded-full blur-2xl group-hover:bg-blue-500/20 transition-colors" />
          <h4 className="text-sm font-medium text-text-primary mb-1 relative z-10 flex items-center gap-2">
            <svg className="w-4 h-4 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" /></svg>
            Audio Overview
          </h4>
          <p className="text-[10px] text-text-secondary leading-relaxed mb-4 relative z-10">
            Generate a deep-dive podcast conversation synthesizing your selected sources.
          </p>
          <button className="w-full py-2 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 text-xs font-medium rounded-lg transition-colors border border-blue-500/30 relative z-10 flex justify-center items-center gap-2 shadow-[0_0_15px_rgba(59,130,246,0.15)]">
            <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z" /></svg>
            Generate Podcast
          </button>
        </div>

        {/* Source Management */}
        <div>
           <div className="flex justify-between items-center mb-3">
             <h4 className="text-[10px] uppercase tracking-widest text-text-muted font-semibold">Active Sources (3)</h4>
             <button className="text-[10px] text-blue-400 hover:text-blue-300 transition-colors">+</button>
           </div>
           
           <div className="space-y-2">
             <div className="p-3 rounded-xl bg-surface-raised border border-white/5 flex items-start gap-3 cursor-pointer hover:border-white/10 transition-colors">
               <input type="checkbox" defaultChecked className="mt-1 accent-blue-500" />
               <div className="flex-1 min-w-0">
                 <div className="text-xs font-medium text-text-primary truncate">Q3_Macro_Report.pdf</div>
                 <div className="text-[10px] text-text-muted">14 Pages • Parsed</div>
               </div>
             </div>
             
             <div className="p-3 rounded-xl bg-surface-raised border border-white/5 flex items-start gap-3 cursor-pointer hover:border-white/10 transition-colors">
               <input type="checkbox" defaultChecked className="mt-1 accent-blue-500" />
               <div className="flex-1 min-w-0">
                 <div className="text-xs font-medium text-text-primary truncate">Competitor Pricing Matrix</div>
                 <div className="text-[10px] text-text-muted">Google Sheets • Syncing...</div>
               </div>
             </div>
             
             <div className="p-3 rounded-xl bg-surface-raised border border-white/5 flex items-start gap-3 cursor-pointer hover:border-white/10 transition-colors opacity-50">
               <input type="checkbox" className="mt-1 accent-blue-500" />
               <div className="flex-1 min-w-0">
                 <div className="text-xs font-medium text-text-primary truncate">Transcript_CEO_Interview.txt</div>
                 <div className="text-[10px] text-text-muted">Deselcted • 4.2k tokens</div>
               </div>
             </div>
           </div>
        </div>

      </div>
    </div>
  );
}
