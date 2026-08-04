export default function CanvasArea({ activeAsset }: { activeAsset: string | null }) {
  return (
    <div className="w-full h-full rounded-2xl border border-white/10 bg-[#0a0a0a] shadow-2xl relative overflow-hidden group flex items-center justify-center">
      {/* Dynamic Grid Background */}
      <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMiIgY3k9IjIiIHI9IjEiIGZpbGw9InJnYmEoMjU1LDI1NSwyNTUsMC4wNykiLz48L3N2Zz4=')] opacity-50" />
      
      {activeAsset ? (
        <img src={activeAsset} alt="Active Canvas Asset" className="max-w-full max-h-full object-contain relative z-10 shadow-2xl" />
      ) : (
        <div className="relative z-10 text-center">
          <div className="w-20 h-20 bg-white/5 border border-white/10 rounded-2xl mx-auto mb-6 flex items-center justify-center">
            <svg className="w-8 h-8 text-white/20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
          </div>
          <h3 className="text-xl font-light text-text-primary mb-2">Canvas Empty</h3>
          <p className="text-sm text-text-muted max-w-sm mx-auto font-light">
            Generate an image or select an asset from the library to begin.
          </p>
        </div>
      )}
    </div>
  );
}
