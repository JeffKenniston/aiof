import { create } from 'zustand'

export type WorkstationType = 'research' | 'professional' | 'engineering' | 'creative' | 'data'

interface WorkstationState {
  activeWorkstation: WorkstationType
  setActiveWorkstation: (ws: WorkstationType) => void
  isSwitcherVisible: boolean
  toggleSwitcher: () => void
}

export const useWorkstationStore = create<WorkstationState>((set) => ({
  activeWorkstation: 'engineering',
  setActiveWorkstation: (ws) => set({ activeWorkstation: ws }),
  isSwitcherVisible: false,
  toggleSwitcher: () => set((state) => ({ isSwitcherVisible: !state.isSwitcherVisible })),
}))
