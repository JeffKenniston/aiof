import { useLayoutStore } from '@aiof/rxdb-store';
import { motion, AnimatePresence } from 'framer-motion';
import { BrainCircuit, X } from 'lucide-react';
import Chat from '../panels/Chat';
import {} from '@aiof/rxdb-store';

export default function GlobalArchimedesChat() {
  const isOpen = useLayoutStore(state => state.isGlobalChatOpen);
  const setIsOpen = useLayoutStore(state => state.setGlobalChatOpen);

  return (
    <div className="fixed inset-y-0 right-0 z-50 pointer-events-none flex flex-col justify-end">
      {/* Sliding Chat Panel */}
      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ x: '100%', opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: '100%', opacity: 0 }}
            transition={{ type: 'spring', damping: 25, stiffness: 200 }}
            className="absolute top-4 right-4 bottom-20 w-[500px] bg-neutral-900/95 backdrop-blur-2xl border border-white/10 rounded-2xl shadow-2xl flex flex-col overflow-hidden pointer-events-auto"
          >
            <div className="flex-1 overflow-hidden relative flex flex-col">
              <Chat panelId="global-archimedes" transparentBg={true} hideHeader={false} />
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Floating Action Button */}
      <div className="pointer-events-auto absolute bottom-6 right-6">
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={() => setIsOpen(!isOpen)}
          className="w-14 h-14 rounded-full bg-accent-blue/90 hover:bg-accent-blue text-white shadow-[0_0_20px_rgba(59,130,246,0.4)] hover:shadow-[0_0_30px_rgba(59,130,246,0.6)] backdrop-blur-md flex items-center justify-center border border-white/20 transition-shadow"
        >
          {isOpen ? <X className="w-6 h-6" /> : <BrainCircuit className="w-6 h-6" />}
        </motion.button>
      </div>
    </div>
  );
}
