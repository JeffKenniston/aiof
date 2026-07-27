import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type WorkstationType = 'education' | 'research' | 'professional' | 'engineering' | 'creative' | 'data' | 'chat' | 'fabricator'

interface WorkstationState {
  activeWorkstation: WorkstationType
  setActiveWorkstation: (type: WorkstationType) => void
  isSwitcherVisible: boolean
  toggleSwitcher: () => void
}

export const useWorkstationStore = create<WorkstationState>()(
  persist(
    (set) => ({
      activeWorkstation: 'fabricator',
      setActiveWorkstation: (type) => set({ activeWorkstation: type }),
      isSwitcherVisible: false,
      toggleSwitcher: () => set((state) => ({ isSwitcherVisible: !state.isSwitcherVisible })),
    }),
    {
      name: 'workstation-storage',
    }
  )
)
