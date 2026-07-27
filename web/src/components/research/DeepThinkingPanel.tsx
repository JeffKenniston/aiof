
import { motion } from 'framer-motion';

export default function DeepThinkingPanel() {
  return (
    <div className="bg-surface-raised rounded-2xl p-5 border border-white/5 shadow-inner relative overflow-hidden group">
      <div className="absolute inset-0 bg-gradient-to-br from-white/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none" />
      
      <h4 className="text-xs font-medium text-text-muted uppercase tracking-wider mb-5 flex items-center gap-2">
        <div className="w-2 h-2 rounded-full bg-accent-purple animate-pulse shadow-[0_0_8px_rgba(189,147,249,0.6)]" />
        Agent Thought Process
      </h4>
      <div className="space-y-4 relative">
        {/* Animated steps */}
        <motion.div initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }} className="flex gap-3 relative">
          <div className="w-px bg-gradient-to-b from-accent-purple/50 to-transparent ml-[11px] mt-6 absolute h-8" />
          <div className="w-6 h-6 rounded-full bg-accent-purple/10 flex items-center justify-center shrink-0 z-10 border border-accent-purple/30 backdrop-blur-md shadow-sm">
            <span className="text-accent-purple text-[10px] font-bold">1</span>
          </div>
          <p className="text-sm text-text-secondary pt-0.5 font-light">Analyzing query intent and identifying research domains...</p>
        </motion.div>
        
        {/* Step 2 */}
        <motion.div initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.2 }} className="flex gap-3 relative">
          <div className="w-px bg-gradient-to-b from-accent-blue/50 to-transparent ml-[11px] mt-6 absolute h-8" />
          <div className="w-6 h-6 rounded-full bg-accent-blue/10 flex items-center justify-center shrink-0 z-10 border border-accent-blue/30 backdrop-blur-md shadow-sm">
            <span className="text-accent-blue text-[10px] font-bold">2</span>
          </div>
          <p className="text-sm text-text-secondary pt-0.5 font-light">Formulating sub-queries for deep semantic search</p>
        </motion.div>
        
        {/* Step 3 (Simulated loading) */}
        <motion.div initial={{ opacity: 0, x: -10 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.4 }} className="flex gap-3 relative">
          <div className="w-6 h-6 rounded-full bg-surface-glass flex items-center justify-center shrink-0 z-10 border border-white/10 backdrop-blur-md shadow-sm">
            <span className="w-1 h-1 rounded-full bg-text-muted animate-ping" />
          </div>
          <p className="text-sm text-text-muted pt-0.5 font-light italic">Synthesizing context from 14 source nodes...</p>
        </motion.div>
      </div>
    </div>
  );
}
