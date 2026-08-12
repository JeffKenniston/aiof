

export default function CitationCard({ title, url, snippet }: { title: string, url: string, snippet: string }) {
  return (
    <a 
      href={url} 
      target="_blank" 
      rel="noreferrer" 
      className="block p-4 rounded-2xl bg-surface-raised border border-white/5 hover:border-accent-blue/50 hover:bg-surface-glass hover:shadow-[0_8px_30px_rgb(0,0,0,0.12)] transition-all duration-300 group cursor-pointer"
    >
      <h5 className="font-medium text-text-primary text-sm group-hover:text-accent-blue transition-colors line-clamp-1">{title}</h5>
      <p className="text-xs text-text-muted mt-2 line-clamp-2 leading-relaxed font-light">{snippet}</p>
      <div className="mt-4 flex items-center gap-2 text-[10px] text-accent-blue/50 uppercase tracking-wider font-semibold truncate group-hover:text-accent-blue/80 transition-colors">
        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
        <span className="truncate">{url}</span>
      </div>
    </a>
  );
}
