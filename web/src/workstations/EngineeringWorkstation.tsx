import { useEffect } from 'react'
import AppShell from '../layout/AppShell'
import PanelGrid from '../layout/PanelGrid'
import { getDatabase } from '../db'
import { startReplication } from '../db/replication'
import { useBreakpoint } from '../hooks/useBreakpoint'
import { useLayoutStore } from '../hooks/useLayoutStore'
import PanelShell from '../layout/PanelShell'

export default function EngineeringWorkstation() {
  const breakpoint = useBreakpoint();
  const { mobileActivePanel, activeProject, viewModeOverride, setWorkspaceId } = useLayoutStore();
  const resolvedViewMode = viewModeOverride === 'auto' ? breakpoint : viewModeOverride;
  
  // Engineering always uses the 'dev' workspace preset
  useEffect(() => {
    setWorkspaceId('dev');
  }, [setWorkspaceId]);

  useEffect(() => {
    if (!activeProject) return;

    const controller = new AbortController();
    
    getDatabase(activeProject).then(() => {
      startReplication(activeProject, controller.signal);
    });

    return () => {
      controller.abort();
    };
  }, [activeProject]);

  if (!activeProject) {
    return (
      <AppShell>
        <div className="w-full h-full flex items-center justify-center text-text-muted">
          <div className="flex flex-col items-center gap-6">
            <div className="w-10 h-10 rounded-full border-2 border-accent-blue/40 border-t-transparent animate-spin"></div>
            <span className="text-sm font-medium tracking-wide text-text-secondary">Initializing workspace…</span>
          </div>
        </div>
      </AppShell>
    );
  }

  return (
    <AppShell>
      <div 
        className={`engineering-workspace w-full h-full flex flex-col relative transition-all duration-500 ease-out gpu-accelerated ${
          resolvedViewMode === 'condensed' ? 'compact-stack p-0' : 
          resolvedViewMode === 'semi-full' ? 'medium-split' : 
          resolvedViewMode === 'full' ? 'full-grid' : 'expanded-grid'
        }`}
        key={`${resolvedViewMode}-${activeProject}`}
      >
        {resolvedViewMode === 'condensed' ? (
          <div className="w-full h-full">
            <PanelShell panelId={mobileActivePanel} />
          </div>
        ) : (
          <PanelGrid />
        )}
      </div>
    </AppShell>
  )
}
