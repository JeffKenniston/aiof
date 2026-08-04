import { useEffect } from 'react'
import AppShell from '../layout/AppShell'
import PanelGrid from '../layout/PanelGrid'

import { useBreakpoint } from '../hooks/useBreakpoint'
import { useLayoutStore } from '../hooks/useLayoutStore'
import PanelShell from '../layout/PanelShell'

import SpatialCanvas from '../components/SpatialCanvas'
import ContextRings from '../components/ContextRings'

export default function EngineeringWorkstation() {
  const breakpoint = useBreakpoint();
  const { mobileActivePanel, activeProject, viewModeOverride, setWorkspaceId } = useLayoutStore();
  const resolvedViewMode = viewModeOverride === 'auto' ? breakpoint : viewModeOverride;
  
  // Engineering always uses the 'dev' workspace preset
  useEffect(() => {
    setWorkspaceId('dev');
  }, [setWorkspaceId]);

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
        className={`engineering-workspace w-full h-full flex flex-col relative transition-all duration-500 ease-out gpu-accelerated`}
        key={`${resolvedViewMode}-${activeProject}`}
      >
        {resolvedViewMode === 'condensed' ? (
          <div className="w-full h-full">
            <PanelShell panelId={mobileActivePanel} />
          </div>
        ) : (
          <div className="flex w-full h-full relative overflow-hidden">
            {/* Left Surface: Spatial Canvas */}
            <div className="w-1/2 h-full">
              <SpatialCanvas />
            </div>
            {/* Right Surface: Orchestration Grid & Telemetry */}
            <div className="w-1/2 h-full flex bg-surface-base relative">
              <div className="flex-1 relative">
                 <PanelGrid />
              </div>
              <div className="w-64 border-l border-border-default h-full bg-surface-base z-10">
                 <ContextRings />
              </div>
            </div>
          </div>
        )}
      </div>
    </AppShell>
  )
}
