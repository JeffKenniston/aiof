import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import Chat from '../panels/Chat';
import DeepThinkingPanel from '../components/research/DeepThinkingPanel';
import CitationCard from '../components/research/CitationCard';
import KnowledgeVaultSidebar from '../components/research/KnowledgeVaultSidebar';

export default function ResearchWorkstation() {
  const [leftOpen, setLeftOpen] = useState(true);
  const [rightOpen, setRightOpen] = useState(true);

  return (
    <div className="w-full h-full flex bg-[#030305] text-text-primary overflow-hidden relative font-sans">
      {/* Background gradients */}
      <div className="absolute top-0 left-1/4 w-[600px] h-[400px] bg-blue-600/5 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-0 right-1/4 w-[600px] h-[400px] bg-purple-600/5 rounded-full blur-[120px] pointer-events-none" />

      {/* Left Pane: Knowledge Vault */}
      <AnimatePresence initial={false}>
        {leftOpen && (
          <motion.div 
            initial={{ width: 0, opacity: 0, x: -50 }}
            animate={{ width: 300, opacity: 1, x: 0 }}
            exit={{ width: 0, opacity: 0, x: -50 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="h-full border-r border-white/5 bg-[#0a0a0c]/80 backdrop-blur-3xl z-20 shrink-0 shadow-[10px_0_30px_rgba(0,0,0,0.5)]"
          >
            <KnowledgeVaultSidebar onClose={() => setLeftOpen(false)} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Center Pane: Research Canvas */}
      <div className="flex-1 flex flex-col h-full relative z-10">
        <header className="h-14 flex items-center justify-between px-4 absolute top-0 left-0 right-0 z-30 pointer-events-none">
          <div className="flex items-center gap-3 pointer-events-auto">
            {!leftOpen && (
              <button onClick={() => setLeftOpen(true)} className="p-1.5 text-text-muted hover:text-white transition-colors bg-white/5 rounded backdrop-blur-md shadow-lg border border-white/10">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" /></svg>
              </button>
            )}
          </div>
          <div className="pointer-events-auto">
            {!rightOpen && (
              <button onClick={() => setRightOpen(true)} className="p-1.5 text-text-muted hover:text-white transition-colors bg-white/5 rounded backdrop-blur-md shadow-lg border border-white/10">
                 <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
              </button>
            )}
          </div>
        </header>

        <div className="flex-1 max-w-4xl mx-auto w-full pt-14 pb-8 flex flex-col">
           {/* Passing transparentBg to blend perfectly with our premium background */}
           <div className="flex-1 border border-white/5 bg-black/20 rounded-3xl overflow-hidden shadow-2xl backdrop-blur-sm">
             <Chat panelId="research-chat" hideHeader={true} transparentBg={true} />
           </div>
        </div>
      </div>

      {/* Right Pane: Intelligence Inspector (Deep Thinking + Citations) */}
      <AnimatePresence initial={false}>
        {rightOpen && (
          <motion.div 
            initial={{ width: 0, opacity: 0, x: 50 }}
            animate={{ width: 340, opacity: 1, x: 0 }}
            exit={{ width: 0, opacity: 0, x: 50 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="h-full border-l border-white/5 bg-[#0a0a0c]/80 backdrop-blur-3xl z-20 shrink-0 flex flex-col shadow-[-10px_0_30px_rgba(0,0,0,0.5)]"
          >
            <div className="h-14 flex items-center justify-between px-4 border-b border-white/5 bg-white/[0.01]">
              <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Intelligence Inspector</span>
              <button onClick={() => setRightOpen(false)} className="text-text-muted hover:text-white transition-colors">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
            
            <div className="flex-1 overflow-y-auto p-4 space-y-6">
              <DeepThinkingPanel />
              
              <div>
                <h4 className="text-[10px] font-medium text-text-secondary uppercase tracking-widest mb-4 flex items-center gap-2">
                  <svg className="w-3 h-3 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                  Verified Sources
                </h4>
                <div className="space-y-3">
                  <CitationCard 
                    title="Attention Is All You Need"
                    url="https://arxiv.org/abs/1706.03762"
                    snippet="The dominant sequence transduction models are based on complex recurrent or convolutional neural networks that include an encoder and a decoder. The best performing models also connect the encoder and decoder through an attention mechanism. We propose a new simple network architecture, the Transformer, based solely on attention mechanisms, dispensing with recurrence and convolutions entirely."
                  />
                  <CitationCard 
                    title="Transformer Models in Vision"
                    url="https://arxiv.org/abs/2010.11929"
                    snippet="While the Transformer architecture has become the de-facto standard for natural language processing tasks, its applications to computer vision remain limited. In vision, attention is either applied in conjunction with convolutional networks, or used to replace certain components of convolutional networks while keeping their overall structure in place. We show that this reliance on CNNs is not necessary and a pure transformer applied directly to sequences of image patches can perform very well on image classification tasks."
                  />
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
