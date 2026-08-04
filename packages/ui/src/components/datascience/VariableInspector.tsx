export default function VariableInspector() {
  // A real implementation would pull from a context or a store
  // representing the active kernel's memory space.
  // We'll mock a hook here that represents the "live" state.
  const activeVariables: any[] = []; 

  return (
    <div className="h-full flex flex-col bg-surface-base border-t border-white/5">
      <div className="h-10 flex items-center px-4 border-b border-white/5 bg-white/[0.02]">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Variable Explorer</span>
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-1">
        {activeVariables.length === 0 ? (
          <div className="text-xs text-text-muted text-center mt-4">
            No variables in memory.<br/>Run a cell to populate.
          </div>
        ) : (
          activeVariables.map((v, i) => (
            <div key={i} className="flex justify-between items-center text-xs p-2 rounded hover:bg-white/5">
              <span className="font-mono text-emerald-400">{v.name}</span>
              <span className="text-text-muted">{v.type}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
