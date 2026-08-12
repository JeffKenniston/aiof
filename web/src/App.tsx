import { AnimatePresence, motion } from 'framer-motion'
import { useWorkstationStore } from './stores/useWorkstationStore'
import WorkstationSwitcher from './components/WorkstationSwitcher'

import EngineeringWorkstation from './workstations/EngineeringWorkstation'
import ResearchWorkstation from './workstations/ResearchWorkstation'
import ProfessionalWorkstation from './workstations/ProfessionalWorkstation'
import CreativeWorkstation from './workstations/CreativeWorkstation'
import DataScienceWorkstation from './workstations/DataScienceWorkstation'

function App() {
  const activeWorkstation = useWorkstationStore(state => state.activeWorkstation)

  const renderWorkstation = () => {
    switch (activeWorkstation) {
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
      default:
        return null
    }
  }

  return (
    <div className="w-screen h-screen bg-neutral-950 text-neutral-50 flex flex-col overflow-hidden font-sans">
      <WorkstationSwitcher />
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
  )
}

export default App
