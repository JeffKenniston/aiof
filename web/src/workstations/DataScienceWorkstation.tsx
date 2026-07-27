import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import NotebookCanvas from '../components/datascience/NotebookCanvas';
import DatasetSidebar from '../components/datascience/DatasetSidebar';
import VariableInspector from '../components/datascience/VariableInspector';

export default function DataScienceWorkstation() {
  const [leftOpen, setLeftOpen] = useState(true);
  const [rightOpen, setRightOpen] = useState(false);

  return (
    <div className="w-full h-full flex bg-[#030303] text-text-primary overflow-hidden relative font-sans">
      {/* Background gradients for data science theme (Deep teal / emerald) */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[300px] bg-emerald-500/5 rounded-[100%] blur-[100px] pointer-events-none" />

      {/* Left Pane: Sources & Datasets */}
      <AnimatePresence initial={false}>
        {leftOpen && (
          <motion.div 
            initial={{ width: 0, opacity: 0 }}
            animate={{ width: 280, opacity: 1 }}
            exit={{ width: 0, opacity: 0 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="h-full border-r border-white/5 bg-surface-base/30 backdrop-blur-3xl z-20 shrink-0"
          >
            <DatasetSidebar onClose={() => {}} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Center Pane: Notebook Canvas */}
      <div className="flex-1 flex flex-col h-full relative z-10">
        <header className="h-14 border-b border-white/5 flex items-center justify-between px-4 bg-white/[0.01]">
          <div className="flex items-center gap-3">
            {!leftOpen && (
              <button onClick={() => setLeftOpen(true)} className="p-1.5 text-text-muted hover:text-white transition-colors bg-white/5 rounded">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" /></svg>
              </button>
            )}
            <h2 className="text-sm font-medium tracking-wide flex items-center gap-2">
              <span className="text-emerald-400">📊</span> Global_Macro_Analysis.ipynb
            </h2>
          </div>
          <button 
            onClick={() => setRightOpen(!rightOpen)} 
            className={`p-1.5 transition-colors rounded ${rightOpen ? 'bg-emerald-500/20 text-emerald-400' : 'text-text-muted hover:text-white hover:bg-white/5'}`}
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" /></svg>
          </button>
        </header>
        
        <div className="flex-1 overflow-y-auto custom-scrollbar bg-black/20">
          <NotebookCanvas />
        </div>
      </div>

      {/* Right Pane: Variable Inspector */}
      <AnimatePresence initial={false}>
        {rightOpen && (
          <motion.div 
            initial={{ width: 0, opacity: 0 }}
            animate={{ width: 320, opacity: 1 }}
            exit={{ width: 0, opacity: 0 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="h-full border-l border-white/5 bg-[#080808]/80 backdrop-blur-3xl z-20 shrink-0 shadow-[-10px_0_30px_rgba(0,0,0,0.5)]"
          >
            <VariableInspector />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
