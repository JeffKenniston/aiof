export default function ExecutiveSummaryFeed() {
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {/* Item 1 */}
      <div className="p-4 rounded-2xl bg-white/5 border border-white/5 hover:bg-white/10 transition-colors cursor-pointer group">
        <div className="flex items-center gap-2 mb-2">
          <span className="px-2 py-0.5 rounded text-[9px] uppercase tracking-wider font-semibold bg-blue-500/20 text-blue-400">Meeting</span>
          <span className="text-[10px] text-text-muted">10:30 AM</span>
        </div>
        <h4 className="text-sm font-medium text-text-primary mb-1 group-hover:text-amber-500 transition-colors">Q3 Product Roadmap Sync</h4>
        <p className="text-xs text-text-secondary leading-relaxed font-light line-clamp-2">
          Engineering is blocked on the new Auth API. Marketing requested a 2-week delay on launch. Action required on resource allocation.
        </p>
      </div>

      {/* Item 2 */}
      <div className="p-4 rounded-2xl bg-white/5 border border-white/5 hover:bg-white/10 transition-colors cursor-pointer group">
        <div className="flex items-center gap-2 mb-2">
          <span className="px-2 py-0.5 rounded text-[9px] uppercase tracking-wider font-semibold bg-emerald-500/20 text-emerald-400">Market Intel</span>
          <span className="text-[10px] text-text-muted">Yesterday</span>
        </div>
        <h4 className="text-sm font-medium text-text-primary mb-1 group-hover:text-amber-500 transition-colors">Competitor X Pricing Change</h4>
        <p className="text-xs text-text-secondary leading-relaxed font-light line-clamp-2">
          Agent scanned 14 news sources. Competitor X reduced enterprise tier pricing by 15%. Recommend reviewing our Q4 retention strategy.
        </p>
      </div>
      
      {/* Item 3 */}
      <div className="p-4 rounded-2xl bg-white/5 border border-white/5 hover:bg-white/10 transition-colors cursor-pointer group">
        <div className="flex items-center gap-2 mb-2">
          <span className="px-2 py-0.5 rounded text-[9px] uppercase tracking-wider font-semibold bg-amber-500/20 text-amber-500">Email Triaged</span>
          <span className="text-[10px] text-text-muted">2 Days Ago</span>
        </div>
        <h4 className="text-sm font-medium text-text-primary mb-1 group-hover:text-amber-500 transition-colors">Contract Negotiation with Vendor Y</h4>
        <p className="text-xs text-text-secondary leading-relaxed font-light line-clamp-2">
          Vendor agreed to the revised SLAs. Legal agent generated the final redlines. Awaiting your signature via DocuSign.
        </p>
      </div>
    </div>
  );
}
