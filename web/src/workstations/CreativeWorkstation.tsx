import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import AssetLibrary from '../components/creative/AssetLibrary';
import CanvasArea from '../components/creative/CanvasArea';
import GenerationPalette from '../components/creative/GenerationPalette';

export default function CreativeWorkstation() {
  const [leftOpen, setLeftOpen] = useState(true);
  const [rightOpen, setRightOpen] = useState(true);

  return (
    <div className="w-full h-full flex bg-[#030105] text-text-primary overflow-hidden relative font-sans">
      {/* Background gradients for creative theme (Fuchsia / Violet) */}
      <div className="absolute top-0 right-1/4 w-[600px] h-[500px] bg-fuchsia-600/5 rounded-[100%] blur-[120px] pointer-events-none" />
      <div className="absolute bottom-1/4 left-1/4 w-[800px] h-[400px] bg-violet-600/5 rounded-[100%] blur-[120px] pointer-events-none" />

      {/* Left Pane: Asset Library */}
      <AnimatePresence initial={false}>
        {leftOpen && (
          <motion.div 
            initial={{ width: 0, opacity: 0, x: -50 }}
            animate={{ width: 280, opacity: 1, x: 0 }}
            exit={{ width: 0, opacity: 0, x: -50 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="h-full border-r border-white/5 bg-[#0a0a0c]/80 backdrop-blur-3xl z-20 shrink-0 shadow-[10px_0_30px_rgba(0,0,0,0.5)]"
          >
            <AssetLibrary onClose={() => setLeftOpen(false)} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Center Pane: Active Canvas */}
      <div className="flex-1 flex flex-col h-full relative z-10">
        <header className="h-14 flex items-center justify-between px-4 absolute top-0 left-0 right-0 z-30">
          <div className="flex items-center gap-3">
            {!leftOpen && (
              <button onClick={() => setLeftOpen(true)} className="p-1.5 text-text-muted hover:text-white transition-colors bg-white/5 rounded backdrop-blur-md shadow-lg border border-white/10">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" /></svg>
              </button>
            )}
          </div>
          {!rightOpen && (
            <button onClick={() => setRightOpen(true)} className="p-1.5 text-text-muted hover:text-white transition-colors bg-white/5 rounded backdrop-blur-md shadow-lg border border-white/10">
               <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" /></svg>
            </button>
          )}
        </header>
        
        <div className="flex-1 bg-transparent flex items-center justify-center p-8 lg:p-12">
          <CanvasArea />
        </div>
      </div>

      {/* Right Pane: Generation Palette */}
      <AnimatePresence initial={false}>
        {rightOpen && (
          <motion.div 
            initial={{ width: 0, opacity: 0, x: 50 }}
            animate={{ width: 340, opacity: 1, x: 0 }}
            exit={{ width: 0, opacity: 0, x: 50 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="h-full border-l border-white/5 bg-[#0a0a0c]/80 backdrop-blur-3xl z-20 shrink-0 shadow-[-10px_0_30px_rgba(0,0,0,0.5)]"
          >
            <GenerationPalette onClose={() => setRightOpen(false)} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
