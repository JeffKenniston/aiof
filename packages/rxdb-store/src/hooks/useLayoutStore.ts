import { create } from 'zustand';
import { PRESETS } from '../workspace/presets';

interface LayoutState {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  rightSidebarCollapsed: boolean;
  toggleRightSidebar: () => void;
  footerCollapsed: boolean;
  toggleFooter: () => void;
  activePanel: string;
  setActivePanel: (id: string) => void;
  mobileActivePanel: string;
  setMobileActivePanel: (id: string) => void;
  activePreset: string;
  activeWorkspaceId: string;
  setWorkspaceId: (id: string) => void;
  viewModeOverride: 'auto' | 'condensed' | 'semi-full' | 'full' | 'expanded';
  setViewModeOverride: (mode: 'auto' | 'condensed' | 'semi-full' | 'full' | 'expanded') => void;
  bottomSheetPanel: string | null;
  setBottomSheetPanel: (id: string | null) => void;
  activeProject: string;
  setActiveProject: (id: string) => void;
  isProjectManagerOpen: boolean;
  setProjectManagerOpen: (isOpen: boolean) => void;
  isGlobalChatOpen: boolean;
  setGlobalChatOpen: (isOpen: boolean) => void;
}

export const useLayoutStore = create<LayoutState>((set) => {
  const initialWorkspace = PRESETS.find(p => p.id === 'casual') || PRESETS[0];
  
  return {
    sidebarCollapsed: false,
    toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
    rightSidebarCollapsed: true,
    toggleRightSidebar: () => set((state) => ({ rightSidebarCollapsed: !state.rightSidebarCollapsed })),
    footerCollapsed: true,
    toggleFooter: () => set((state) => ({ footerCollapsed: !state.footerCollapsed })),
    
    activePanel: '',
    setActivePanel: (id) => set({ activePanel: id }),
    
    mobileActivePanel: initialWorkspace.defaultPanelId,
    setMobileActivePanel: (id) => set({ mobileActivePanel: id }),
    
    activePreset: initialWorkspace.name,
    activeWorkspaceId: initialWorkspace.id,
    setWorkspaceId: (id) => {
      const workspace = PRESETS.find(p => p.id === id);
      if (workspace) {
        set({
          activeWorkspaceId: workspace.id,
          activePreset: workspace.name,
          mobileActivePanel: workspace.defaultPanelId
        });
      }
    },
    
    viewModeOverride: 'auto',
    setViewModeOverride: (mode) => set({ viewModeOverride: mode }),
    
    bottomSheetPanel: null,
    setBottomSheetPanel: (id) => set({ bottomSheetPanel: id }),
    
    activeProject: '',
    setActiveProject: (id) => set({ activeProject: id }),
    
    isProjectManagerOpen: false,
    setProjectManagerOpen: (isOpen) => set({ isProjectManagerOpen: isOpen }),
    
    isGlobalChatOpen: false,
    setGlobalChatOpen: (isOpen) => set({ isGlobalChatOpen: isOpen }),
  };
});
