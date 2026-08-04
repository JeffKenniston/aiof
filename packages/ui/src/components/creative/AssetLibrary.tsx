export default function AssetLibrary({ onClose, assets, onSelectAsset }: { onClose: () => void, assets: string[], onSelectAsset: (url: string) => void }) {
  return (
    <div className="w-full h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-5 border-b border-white/5 shrink-0">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Asset Library</span>
        <button onClick={onClose} className="text-text-muted hover:text-white transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      
      <div className="flex-1 overflow-y-auto p-3">
        <div className="grid grid-cols-2 gap-3">
          {assets.map((assetUrl, i) => (
            <div 
              key={i} 
              onClick={() => onSelectAsset(assetUrl)}
              className="aspect-square rounded-xl bg-surface-base border border-white/5 overflow-hidden group cursor-pointer relative"
            >
              <img src={assetUrl} alt={`Asset ${i}`} className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500" />
              <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex items-end p-2">
                <span className="text-[9px] text-white/80 font-medium truncate">Generated {i+1}</span>
              </div>
            </div>
          ))}
          {assets.length === 0 && (
            <div className="col-span-2 text-center text-text-muted text-xs p-4 border border-dashed border-white/10 rounded-xl">
              No assets yet.<br/>Generate one from the palette.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
