import { Suspense } from 'react';
import { PANEL_REGISTRY } from '../workspace/registry';

interface PanelShellProps {
  panelId: string;
}

export default function PanelShell({ panelId }: PanelShellProps) {
  const def = PANEL_REGISTRY[panelId];
  
  if (!def) {
    return (
      <div className="w-full h-full flex items-center justify-center bg-surface-raised/50 rounded-xl">
        <span className="text-text-muted text-sm">Panel Not Found: {panelId}</span>
      </div>
    );
  }
  
  const Component = def.component;
  
  return (
    <div className="panel-shell w-full h-full flex flex-col bg-surface-raised/80 rounded-xl overflow-hidden backdrop-blur-sm transition-all duration-300">
      {/* Minimal chrome header — thin, unobtrusive */}
      <div className="panel-chrome h-7 border-b border-white/[0.04] flex items-center px-3 select-none shrink-0 cursor-move">
        <span className="mr-1.5 text-[11px] opacity-60">{def.icon}</span>
        <span className="text-[11px] font-medium text-text-muted tracking-widest uppercase">{def.title}</span>
        <div className="flex-1" />
        <button className="w-4 h-4 flex items-center justify-center text-text-muted/40 hover:text-text-primary rounded transition-colors opacity-0 group-hover:opacity-100">
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      
      {/* Content */}
      <div className="flex-1 min-h-0 relative">
        <Suspense fallback={
          <div className="absolute inset-0 flex items-center justify-center bg-surface-base/50">
            <div className="w-5 h-5 border-2 border-accent-blue/30 border-t-accent-blue rounded-full animate-spin" />
          </div>
        }>
          <Component panelId={panelId} />
        </Suspense>
      </div>
    </div>
  );
}
