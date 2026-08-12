export default function GenerationPalette({ onClose }: { onClose: () => void }) {
  return (
    <div className="w-[340px] h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-4 border-b border-white/5 bg-white/[0.01]">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Generation Settings</span>
        <button onClick={onClose} className="text-text-muted hover:text-white transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      
      <div className="flex-1 overflow-y-auto p-5 space-y-8">
        
        {/* Prompt Input */}
        <div className="space-y-3">
          <label className="text-[10px] uppercase tracking-widest text-text-muted font-semibold flex items-center gap-2">
            <svg className="w-3 h-3 text-fuchsia-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
            Prompt
          </label>
          <div className="relative group">
            <div className="absolute -inset-0.5 bg-gradient-to-r from-fuchsia-600 to-violet-600 rounded-xl blur opacity-20 group-hover:opacity-40 transition duration-1000 group-hover:duration-200" />
            <textarea 
              className="relative w-full h-32 bg-black/60 border border-white/10 rounded-xl p-3 text-sm text-text-primary focus:outline-none focus:border-fuchsia-500/50 resize-none transition-colors backdrop-blur-sm"
              placeholder="Describe the image you want to generate in detail..."
            />
          </div>
          <div className="flex justify-between items-center text-[10px] text-text-muted">
            <span>Shift + Enter for new line</span>
            <span>0/1000</span>
          </div>
        </div>

        {/* Aspect Ratio */}
        <div className="space-y-3">
           <label className="text-[10px] uppercase tracking-widest text-text-muted font-semibold">Aspect Ratio</label>
           <div className="grid grid-cols-3 gap-2">
             <button className="aspect-square bg-surface-raised border border-white/5 rounded-xl flex flex-col items-center justify-center gap-2 hover:border-fuchsia-500/50 transition-colors text-text-muted hover:text-white group shadow-sm">
               <div className="w-5 h-5 border-2 border-current rounded-sm group-hover:scale-110 transition-transform" />
               <span className="text-[10px] font-medium">1:1</span>
             </button>
             <button className="aspect-square bg-fuchsia-500/10 border border-fuchsia-500/30 rounded-xl flex flex-col items-center justify-center gap-2 text-fuchsia-400 shadow-[0_0_15px_rgba(192,38,211,0.1)] group">
               <div className="w-6 h-4 border-2 border-current rounded-sm group-hover:scale-110 transition-transform" />
               <span className="text-[10px] font-medium">16:9</span>
             </button>
             <button className="aspect-square bg-surface-raised border border-white/5 rounded-xl flex flex-col items-center justify-center gap-2 hover:border-fuchsia-500/50 transition-colors text-text-muted hover:text-white group shadow-sm">
               <div className="w-4 h-6 border-2 border-current rounded-sm group-hover:scale-110 transition-transform" />
               <span className="text-[10px] font-medium">9:16</span>
             </button>
           </div>
        </div>

        {/* Style Seed */}
        <div className="space-y-3">
          <label className="text-[10px] uppercase tracking-widest text-text-muted font-semibold">Style Reference</label>
          <div className="h-16 rounded-xl border border-white/10 bg-black/40 border-dashed flex items-center justify-center text-text-muted hover:text-fuchsia-400 hover:border-fuchsia-500/50 transition-colors cursor-pointer text-xs group">
            <span className="group-hover:scale-105 transition-transform flex items-center gap-2">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" /></svg>
              Upload Reference Image
            </span>
          </div>
        </div>
        
      </div>

      {/* Generate Button Fixed Bottom */}
      <div className="p-5 border-t border-white/5 bg-white/[0.01]">
        <button className="w-full py-3.5 bg-gradient-to-r from-fuchsia-600 to-violet-600 hover:from-fuchsia-500 hover:to-violet-500 rounded-xl text-white font-medium shadow-[0_0_20px_rgba(192,38,211,0.3)] hover:shadow-[0_0_30px_rgba(192,38,211,0.5)] transition-all flex items-center justify-center gap-2 group">
          <svg className="w-5 h-5 group-hover:rotate-12 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" /></svg>
          Generate
        </button>
      </div>

    </div>
  );
}
