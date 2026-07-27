import { motion, AnimatePresence } from 'framer-motion'
import { BrainCircuit, Briefcase, Code, Paintbrush, LineChart, LayoutGrid, GraduationCap, MessageSquare, Bot } from 'lucide-react'
import { useWorkstationStore } from '../stores/useWorkstationStore'
import type { WorkstationType } from '../stores/useWorkstationStore'

export default function WorkstationSwitcher() {
  const { activeWorkstation, setActiveWorkstation, isSwitcherVisible, toggleSwitcher } = useWorkstationStore()

  const tabs: { id: WorkstationType, label: string, icon: React.ReactNode }[] = [
    { id: 'education', label: 'Education', icon: <GraduationCap className="w-4 h-4" /> },
    { id: 'research', label: 'Research', icon: <BrainCircuit className="w-4 h-4" /> },
    { id: 'professional', label: 'Professional', icon: <Briefcase className="w-4 h-4" /> },
    { id: 'engineering', label: 'Engineering', icon: <Code className="w-4 h-4" /> },
    { id: 'creative', label: 'Creative', icon: <Paintbrush className="w-4 h-4" /> },
    { id: 'data', label: 'Data Science', icon: <LineChart className="w-4 h-4" /> },
    { id: 'fabricator', label: 'Fabricator', icon: <Bot className="w-4 h-4" /> },
    { id: 'chat', label: 'Chat', icon: <MessageSquare className="w-4 h-4" /> }
  ]

  return (
    <div className="absolute top-0 left-0 right-0 z-50 flex flex-col items-center pointer-events-none">
      <AnimatePresence initial={false}>
        {isSwitcherVisible && (
          <motion.div
            initial={{ height: 0, opacity: 0, scale: 0.95 }}
            animate={{ height: 'auto', opacity: 1, scale: 1 }}
            exit={{ height: 0, opacity: 0, scale: 0.95 }}
            transition={{ type: 'spring', damping: 22, stiffness: 350 }}
            className="pointer-events-auto overflow-hidden origin-top"
          >
            <div className="bg-neutral-900/90 backdrop-blur-xl p-1.5 rounded-b-2xl border-x border-b border-white/10 shadow-2xl flex items-center gap-1">
              {tabs.map(tab => {
                const isActive = activeWorkstation === tab.id
                return (
                  <button
                    key={tab.id}
                    onClick={() => {
                      setActiveWorkstation(tab.id)
                      toggleSwitcher()
                    }}
                    className={`relative flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-xl transition-colors outline-none
                      ${isActive ? 'text-white' : 'text-neutral-400 hover:text-neutral-200'}
                    `}
                  >
                    {isActive && (
                      <motion.div
                        layoutId="active-workstation-tab"
                        className="absolute inset-0 bg-white/10 rounded-xl"
                        transition={{ type: "spring", bounce: 0.2, duration: 0.6 }}
                      />
                    )}
                    <span className="relative z-10 flex items-center gap-2">
                      {tab.icon}
                      {tab.label}
                    </span>
                  </button>
                )
              })}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
      
      {/* Prominent trigger button that sits at the top edge */}
      <motion.button 
        layout
        transition={{ type: 'spring', damping: 22, stiffness: 350 }}
        onClick={toggleSwitcher}
        title="Switch Workstation"
        className="pointer-events-auto mt-0 px-5 py-1.5 bg-neutral-900/80 hover:bg-accent-blue hover:text-white backdrop-blur-md border border-t-0 border-white/15 rounded-b-xl text-neutral-300 shadow-md transition-colors duration-300 flex items-center justify-center gap-2 group cursor-pointer"
      >
        <LayoutGrid className="w-4 h-4" />
        <span className="text-xs font-semibold tracking-wide uppercase opacity-0 group-hover:opacity-100 hidden group-hover:block transition-opacity">Workspaces</span>
      </motion.button>
    </div>
  )
}
