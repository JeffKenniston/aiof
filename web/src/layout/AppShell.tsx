import React, { useEffect } from 'react';
import { BotMessageSquare } from 'lucide-react';
import { useBreakpoint } from '../hooks/useBreakpoint';
import { useLayoutStore } from '../hooks/useLayoutStore';

import CommandBar from './CommandBar';
import StatusBar from './StatusBar';
import BottomSheet from './BottomSheet';
import PanelShell from './PanelShell';
import ProjectManagerModal from './ProjectManagerModal';
import ContextMenu from '../components/ContextMenu';
import { PRESETS } from '../workspace/presets';
import { PANEL_NAV_METADATA } from './Sidebar';

export default function AppShell({ children }: { children: React.ReactNode }) {
  const breakpoint = useBreakpoint();
  const {
    toggleSidebar,
    activePanel, setActivePanel,
    setMobileActivePanel,
    activeWorkspaceId, setWorkspaceId,
    viewModeOverride,
    bottomSheetPanel, setBottomSheetPanel,
    isGlobalChatOpen, setGlobalChatOpen
  } = useLayoutStore();

  const resolvedViewMode = viewModeOverride === 'auto' ? breakpoint : viewModeOverride;
  const currentWorkspace = PRESETS.find(w => w.id === activeWorkspaceId) || PRESETS[0];
  const availablePanelIds = currentWorkspace.availablePanels;

  const handleNavigate = (panelId: string) => {
    if (resolvedViewMode === 'condensed') {
      setMobileActivePanel(panelId);
      setBottomSheetPanel(null); // Close sheet if open
    } else {
      setActivePanel(panelId);
    }
  };

  const handlePresetChange = (id: string) => {
    setWorkspaceId(id);
  };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
      const cmdOrCtrl = isMac ? e.metaKey : e.ctrlKey;

      if (cmdOrCtrl && e.key.toLowerCase() === 'p') {
        e.preventDefault();
        if (e.shiftKey) {
          useLayoutStore.getState().setProjectManagerOpen(true);
        } else {
          window.dispatchEvent(new CustomEvent('focus-search'));
        }
      } else if (cmdOrCtrl && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        toggleSidebar();
      } else if (cmdOrCtrl && e.key.toLowerCase() === 'e') {
        e.preventDefault();
        handleNavigate('file-explorer');
      } else if (cmdOrCtrl && e.key.toLowerCase() === 'j') {
        e.preventDefault();
        handleNavigate('terminal');
      } else if (cmdOrCtrl && e.key.toLowerCase() === 'm' && e.shiftKey) {
        e.preventDefault();
        handleNavigate('telemetry');
      } else if (cmdOrCtrl && e.key === ',') {
        e.preventDefault();
        handleNavigate('settings');
      } else if (e.key === 'Escape') {
        if (useLayoutStore.getState().isProjectManagerOpen) {
          useLayoutStore.getState().setProjectManagerOpen(false);
        } else if (useLayoutStore.getState().activePanel === 'settings') {
          handleNavigate('agent-visual'); // return to dashboard
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [toggleSidebar]);

  return (
    <div 
      className="flex w-full h-full overflow-hidden bg-surface-base text-text-primary font-sans antialiased"
      onContextMenu={(e) => e.preventDefault()}
    >
      {/* Main Content Area */}
      <div className={`flex-1 flex flex-col min-w-0 ${resolvedViewMode === 'condensed' ? 'pb-16' : ''}`}>
        
        {/* Command Bar */}
        <CommandBar currentPreset={activeWorkspaceId} onPresetChange={handlePresetChange} />
        
        {/* Workspace / PanelGrid */}
        <div className="flex-1 relative overflow-hidden bg-surface-base">
          <ProjectManagerModal />
          {children}
          
          {resolvedViewMode !== 'condensed' && activePanel === 'settings' && (
            <div className="absolute inset-0 z-50 bg-surface-base flex flex-col animate-in fade-in zoom-in-95 duration-200">
              <div className="h-12 border-b border-border-default flex items-center justify-between px-4 shrink-0 bg-surface-raised">
                <h2 className="font-medium text-text-primary flex items-center gap-2">
                  <svg className="w-5 h-5 text-accent-blue" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                  Workstation Settings
                </h2>
                <button 
                  className="p-1.5 text-text-muted hover:text-text-primary hover:bg-surface-overlay rounded transition-colors"
                  onClick={() => setActivePanel('')}
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
              <div className="flex-1 overflow-hidden">
                <PanelShell panelId="settings" />
              </div>
            </div>
          )}
        </div>
        
        {/* Status Bar */}
        {resolvedViewMode !== 'condensed' && <StatusBar />}
      </div>

      <ContextMenu />

      {/* Mobile Bottom Sheet Overlay */}
      {resolvedViewMode === 'condensed' && bottomSheetPanel === 'more-nav' && (
        <BottomSheet isOpen={true} onClose={() => setBottomSheetPanel(null)} title="More Tools">
          <div className="grid grid-cols-3 gap-4">
            {availablePanelIds.slice(4).map((panelId) => {
              const item = PANEL_NAV_METADATA[panelId] || { label: panelId, icon: <span>📦</span> };
              return (
                <button 
                  key={panelId}
                  className="flex flex-col items-center p-4 bg-surface-overlay rounded-xl border border-border-subtle" 
                  onClick={() => handleNavigate(panelId)}
                >
                  <div className="w-8 h-8 mb-2 flex items-center justify-center text-text-muted">
                    {item.icon}
                  </div>
                  <span className="text-sm font-medium">{item.label}</span>
                </button>
              );
            })}
          </div>
        </BottomSheet>
      )}

    </div>
  );
}
