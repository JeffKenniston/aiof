
import { useLayoutStore } from '../hooks/useLayoutStore';
import { useBreakpoint } from '../hooks/useBreakpoint';
import { PRESETS } from '../workspace/presets';
import PanelShell from './PanelShell';

export default function PanelGrid() {
  const { activeWorkspaceId, viewModeOverride } = useLayoutStore();
  const breakpoint = useBreakpoint();
  
  const resolvedViewMode = viewModeOverride === 'auto' ? breakpoint : viewModeOverride;
  // If condensed mode falls back here, render the semi-full layout
  const layoutMode = resolvedViewMode === 'condensed' ? 'semi-full' : resolvedViewMode;
  
  const currentWorkspace = PRESETS.find(w => w.id === activeWorkspaceId) || PRESETS[0];
  const { cols, rows, cells } = currentWorkspace.layouts[layoutMode];
  
  return (
    <div 
      className="panel-grid w-full h-full p-3 bg-surface-base relative grid transition-all duration-500 ease-out gpu-accelerated"
      style={{
        gap: '10px',
        gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))`,
        gridTemplateRows: `repeat(${rows}, minmax(0, 1fr))`,
      }}
    >
      {cells.map((cell) => (
        <div 
          key={`${currentWorkspace.id}-${cell.panelId}`}
          className="group w-full h-full transition-all duration-500 ease-out"
          style={{
            gridColumn: `${cell.col + 1} / span ${cell.colSpan}`,
            gridRow: `${cell.row + 1} / span ${cell.rowSpan}`,
          }}
        >
          <PanelShell panelId={cell.panelId} />
        </div>
      ))}
    </div>
  );
}


