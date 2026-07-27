import { useEffect } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useWorkstationStore } from './stores/useWorkstationStore'
import { useLayoutStore } from './hooks/useLayoutStore'
import { getDatabase } from './db'
import { startReplication } from './db/replication'
import WorkstationSwitcher from './components/WorkstationSwitcher'
import { BotMessageSquare } from 'lucide-react'
import PanelShell from './layout/PanelShell'
import EngineeringWorkstation from './workstations/EngineeringWorkstation'
import ResearchWorkstation from './workstations/ResearchWorkstation'
import ProfessionalWorkstation from './workstations/ProfessionalWorkstation'
import CreativeWorkstation from './workstations/CreativeWorkstation'
import DataScienceWorkstation from './workstations/DataScienceWorkstation'
import EducationWorkstation from './workstations/EducationWorkstation'
import ChatWorkstation from './workstations/ChatWorkstation'
import FabricatorWorkstation from './workstations/FabricatorWorkstation'
import { BACKEND_URL } from './config'

function App() {
  const activeWorkstation = useWorkstationStore(state => state.activeWorkstation)
  const activeProject = useLayoutStore(state => state.activeProject)
  const setActiveProject = useLayoutStore(state => state.setActiveProject)
  const setProjectManagerOpen = useLayoutStore(state => state.setProjectManagerOpen)

  // Initialize activeProject globally so all workstations have context
  useEffect(() => {
    fetch(`${BACKEND_URL}/api/projects`)
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch projects');
        return res.json();
      })
      .then(data => {
        if (!activeProject) {
          if (data.active) {
            setActiveProject(data.active);
          } else {
            setProjectManagerOpen(true);
          }
        }
      })
      .catch(console.error);
  }, [activeProject, setActiveProject, setProjectManagerOpen]);

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

  const renderWorkstation = () => {
    switch (activeWorkstation) {
      case 'education':
        return <EducationWorkstation key="edu" />
      case 'engineering':
        return <EngineeringWorkstation key="eng" />
      case 'research':
        return <ResearchWorkstation key="res" />
      case 'professional':
        return <ProfessionalWorkstation key="pro" />
      case 'creative':
        return <CreativeWorkstation key="creative" />
      case 'data':
        return <DataScienceWorkstation key="data" />
      case 'chat':
        return <ChatWorkstation key="chat" />
      case 'fabricator':
        return <FabricatorWorkstation key="fab" />
      default:
        return null
    }
  }

  const isGlobalChatOpen = useLayoutStore(state => state.isGlobalChatOpen)
  const setGlobalChatOpen = useLayoutStore(state => state.setGlobalChatOpen)

  return (
    <div className="w-screen h-screen bg-neutral-950 text-neutral-50 flex flex-col overflow-hidden font-sans">
      <WorkstationSwitcher />
      {/* Hover Trigger Zone to Open Chat */}
      {!isGlobalChatOpen && (
        <div 
          className="absolute left-0 top-0 bottom-0 w-4 z-50 bg-surface-raised/60 hover:bg-surface-raised border-r border-white/10 flex flex-col items-center justify-center cursor-e-resize transition-all group backdrop-blur-sm"
          onMouseEnter={() => setGlobalChatOpen(true)}
          title="Hover to open Agent Chat"
        >
          <BotMessageSquare className="w-3 h-3 text-text-muted group-hover:text-accent-blue transition-colors opacity-60 group-hover:opacity-100 shrink-0" />
          <div className="flex items-center justify-center" style={{ marginTop: '12px', marginBottom: '10px', writingMode: 'vertical-rl', textOrientation: 'upright' }}>
            <span className="text-[9px] font-bold tracking-[0.2em] text-text-muted group-hover:text-accent-blue uppercase opacity-60 group-hover:opacity-100 transition-colors">
              AGENTS
            </span>
          </div>
          <BotMessageSquare className="w-3 h-3 text-text-muted group-hover:text-accent-blue transition-colors opacity-60 group-hover:opacity-100 shrink-0" />
        </div>
      )}

      <div className="flex-1 relative flex overflow-hidden">
        
      {/* Sliding Global Chat on the Left */}
        <div 
          className="shrink-0 bg-surface-raised border-r border-border-default z-40 transition-all duration-300 ease-in-out relative overflow-hidden flex flex-col shadow-[4px_0_24px_rgba(0,0,0,0.5)]"
          style={{ width: isGlobalChatOpen ? '450px' : '0px' }}
          onMouseLeave={() => setGlobalChatOpen(false)}
        >
          <div className="w-[450px] h-full absolute top-0 left-0 flex flex-col bg-surface-raised">
             <PanelShell panelId="chat" />
          </div>
        </div>

        {/* Main Workspace Wrapper */}
        <div className="flex-1 relative overflow-hidden">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeWorkstation}
              initial={{ opacity: 0, y: 10, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -10, scale: 0.98 }}
              transition={{ duration: 0.3, ease: 'easeInOut' }}
              className="absolute inset-0 flex flex-col"
            >
              {renderWorkstation()}
            </motion.div>
          </AnimatePresence>
        </div>
      </div>

    </div>
  )
}

export default App
