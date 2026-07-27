export default function AgentDelegationBoard() {
  return (
    <div className="flex-1 p-6 overflow-x-auto">
      <div className="flex gap-6 h-full min-w-[800px]">
        
        {/* Column 1: Inbox / Pending */}
        <div className="flex-1 flex flex-col gap-4">
          <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wider flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-text-muted" /> Backlog
          </h4>
          <div className="flex-1 rounded-2xl bg-black/20 border border-white/5 p-3 space-y-3">
            <div className="p-3 bg-surface-raised rounded-xl border border-white/5 shadow-sm">
              <div className="text-xs font-medium text-text-primary mb-1">Draft Q4 Board Deck</div>
              <div className="text-[10px] text-text-muted">Waiting on financials</div>
            </div>
            <div className="p-3 bg-surface-raised rounded-xl border border-white/5 shadow-sm">
              <div className="text-xs font-medium text-text-primary mb-1">Schedule Syncs</div>
              <div className="text-[10px] text-text-muted">Need availability</div>
            </div>
          </div>
        </div>

        {/* Column 2: Active Agents */}
        <div className="flex-1 flex flex-col gap-4">
          <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wider flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-amber-500 animate-pulse shadow-[0_0_8px_rgba(245,158,11,0.6)]" /> In Progress
          </h4>
          <div className="flex-1 rounded-2xl bg-amber-500/5 border border-amber-500/10 p-3 space-y-3">
            <div className="p-3 bg-surface-raised rounded-xl border border-amber-500/20 shadow-sm relative overflow-hidden">
              <div className="absolute top-0 left-0 w-1 h-full bg-amber-500" />
              <div className="flex justify-between items-start mb-2">
                <div className="text-xs font-medium text-text-primary">Vendor Contract Review</div>
                <div className="w-5 h-5 rounded-full bg-amber-500/20 text-amber-500 flex items-center justify-center text-[10px] font-bold">L</div>
              </div>
              <div className="text-[10px] text-amber-500/80 italic">Scanning indemnity clauses...</div>
            </div>
            <div className="p-3 bg-surface-raised rounded-xl border border-amber-500/20 shadow-sm relative overflow-hidden">
              <div className="absolute top-0 left-0 w-1 h-full bg-amber-500" />
              <div className="flex justify-between items-start mb-2">
                <div className="text-xs font-medium text-text-primary">Competitor Analysis</div>
                <div className="w-5 h-5 rounded-full bg-blue-500/20 text-blue-500 flex items-center justify-center text-[10px] font-bold">R</div>
              </div>
              <div className="text-[10px] text-amber-500/80 italic">Scraping pricing pages...</div>
            </div>
          </div>
        </div>

        {/* Column 3: Review Required */}
        <div className="flex-1 flex flex-col gap-4">
          <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wider flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-blue-500" /> Needs Review
          </h4>
          <div className="flex-1 rounded-2xl bg-black/20 border border-white/5 p-3 space-y-3">
            <div className="p-3 bg-surface-raised rounded-xl border border-blue-500/30 shadow-sm cursor-pointer hover:border-blue-500/50 transition-colors">
              <div className="flex justify-between items-start mb-2">
                <div className="text-xs font-medium text-text-primary">Weekly Social Posts</div>
                <div className="w-5 h-5 rounded-full bg-purple-500/20 text-purple-500 flex items-center justify-center text-[10px] font-bold">M</div>
              </div>
              <div className="text-[10px] text-text-muted">Drafted 5 tweets for approval</div>
              <button className="mt-3 w-full py-1.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30 text-[10px] font-medium hover:bg-blue-500/30 transition-colors">Review Drafts</button>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
