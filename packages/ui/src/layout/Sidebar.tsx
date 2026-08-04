import { useLayoutStore, useBreakpoint } from '@aiof/rxdb-store';
import React from 'react';
import {} from '@aiof/rxdb-store';
import {} from '@aiof/rxdb-store';
import { PRESETS } from '../workspace/presets';

interface SidebarProps {
  activePanel: string;
  onNavigate: (panelId: string) => void;
  collapsed: boolean;
  onToggleCollapse: () => void;
}

export const PANEL_NAV_METADATA: Record<string, { label: string; icon: React.ReactNode }> = {
  'agent-visual': { 
    label: 'Dashboard', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" /></svg> 
  },
  'code-editor': { 
    label: 'Code', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" /></svg> 
  },
  'terminal': { 
    label: 'Terminal', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg> 
  },
  'agent-text': { 
    label: 'Logs', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg> 
  },
  'file-explorer': { 
    label: 'Files', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" /></svg> 
  },
  'telemetry': { 
    label: 'Telemetry', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" /></svg> 
  },
  'settings': { 
    label: 'Settings', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg> 
  },
  'knowledge-graph': { 
    label: 'Graph', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" /></svg> 
  },
  'prompt-playground': { 
    label: 'Playground', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 00-3.86.517L6.05 16.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" /></svg> 
  },
  'task-queue': { 
    label: 'Tasks', 
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" /></svg> 
  },
  'sandbox-monitor': {
    label: 'Sandbox',
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" /></svg>
  },
  'diff-review': {
    label: 'Review',
    icon: <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" /></svg>
  }
};

export default function Sidebar({ activePanel, onNavigate, collapsed, onToggleCollapse }: SidebarProps) {
  const breakpoint = useBreakpoint();
  const { activeWorkspaceId, viewModeOverride } = useLayoutStore();
  
  const resolvedViewMode = viewModeOverride === 'auto' ? breakpoint : viewModeOverride;
  const currentWorkspace = PRESETS.find(w => w.id === activeWorkspaceId) || PRESETS[0];
  const availablePanelIds = currentWorkspace.availablePanels;

  const desktopNavIds = ['agent-visual', 'code-editor', 'terminal', 'agent-text', 'file-explorer', 'telemetry', 'settings'];

  // Condensed / Phone View (Bottom Tab Navigation)
  if (resolvedViewMode === 'condensed') {
    return (
      <div className="fixed bottom-0 left-0 right-0 h-16 bg-surface-raised border-t border-border-default z-sidebar flex flex-row items-center justify-around px-2 pb-safe select-none">
        {availablePanelIds.slice(0, 4).map((panelId) => {
          const item = PANEL_NAV_METADATA[panelId] || { label: panelId, icon: <span>📦</span> };
          const isActive = activePanel === panelId;
          return (
            <button
              key={panelId}
              data-panel-target={panelId}
              className={`flex flex-col items-center justify-center w-16 h-full space-y-0.5 transition-all duration-200 ease-out gpu-accelerated relative ${
                isActive ? 'text-accent-blue scale-105' : 'text-text-muted hover:text-text-primary'
              }`}
              onClick={() => onNavigate(panelId)}
            >
              {item.icon}
              <span className="text-[10px] font-semibold tracking-wide">{item.label}</span>
              {isActive && <div className="absolute top-0 w-8 h-0.5 bg-accent-blue rounded-b-full shadow-[0_1px_8px_rgba(59,130,246,0.5)]"></div>}
            </button>
          );
        })}
        {availablePanelIds.length > 4 && (
          <button
            className="flex flex-col items-center justify-center w-16 h-full space-y-0.5 transition-colors text-text-muted hover:text-text-primary relative"
            onClick={() => useLayoutStore.getState().setBottomSheetPanel('more-nav')}
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
            </svg>
            <span className="text-[10px] font-semibold tracking-wide">More</span>
          </button>
        )}
      </div>
    );
  }

  // Tablet, Desktop, & Immersive View (Left Rail)
  const isExpanded = resolvedViewMode === 'expanded' && !collapsed;
  
  return (
    <div className={`h-full bg-surface-glass backdrop-blur-xl border-r border-border-default z-sidebar flex flex-col transition-all duration-300 ease-out gpu-accelerated ${
      isExpanded ? 'w-60' : 'w-14'
    }`}>
      {/* Workspace Brand Badge */}
      <div className="h-12 flex items-center justify-center border-b border-border-subtle shrink-0">
        {isExpanded ? (
          <div className="flex flex-col items-center">
            <span className="font-bold text-text-primary tracking-wider flex items-center gap-1.5">
              <span>{currentWorkspace.icon}</span>
              <span>AIOF</span>
            </span>
            <span className="text-[9px] text-text-muted tracking-widest uppercase font-mono">{currentWorkspace.name} mode</span>
          </div>
        ) : (
          <span className="font-bold text-text-primary text-sm flex items-center justify-center">
            {currentWorkspace.icon}
          </span>
        )}
      </div>
      
      {/* Scrollable Navigation Area */}
      <div className="flex-1 py-4 flex flex-col gap-1.5 overflow-y-auto overflow-x-hidden">
        {desktopNavIds.map((panelId) => {
          const item = PANEL_NAV_METADATA[panelId] || { label: panelId, icon: <span>📦</span> };
          const isActive = activePanel === panelId;
          return (
            <button
              key={panelId}
              data-panel-target={panelId}
              className={`group flex items-center h-10 w-full relative transition-all duration-200 ease-out gpu-accelerated ${
                isActive ? 'text-accent-blue bg-accent-blue/10 font-semibold' : 'text-text-muted hover:text-text-primary hover:bg-surface-overlay'
              }`}
              onClick={() => onNavigate(panelId)}
              title={!isExpanded ? item.label : undefined}
            >
              {isActive && <div className="absolute left-0 top-0 bottom-0 w-1 bg-accent-blue rounded-r-full shadow-[1px_0_8px_rgba(59,130,246,0.5)]"></div>}
              <div className={`flex items-center justify-center shrink-0 transition-transform group-hover:scale-105 duration-200 ${isExpanded ? 'w-14' : 'w-full'}`}>
                {item.icon}
              </div>
              {isExpanded && <span className="text-sm tracking-wide whitespace-nowrap">{item.label}</span>}
            </button>
          );
        })}
      </div>
      
      {/* Collapse Trigger Rail (only in Expanded workstation size) */}
      {resolvedViewMode === 'expanded' && (
        <div className="h-12 border-t border-border-subtle flex items-center justify-center shrink-0">
          <button
            className="w-full h-full flex items-center justify-center text-text-muted hover:text-text-primary transition-colors hover:bg-surface-overlay cursor-pointer"
            onClick={onToggleCollapse}
          >
            <svg className={`w-5 h-5 transition-transform duration-300 ${collapsed ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
            </svg>
          </button>
        </div>
      )}
    </div>
  );
}
