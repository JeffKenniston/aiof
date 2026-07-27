import { useState } from 'react';

export default function GenerationPalette({ onClose, onGenerate }: { onClose: () => void, onGenerate: (url: string) => void }) {
  const [prompt, setPrompt] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);

  const handleGenerate = async () => {
    if (!prompt.trim()) return;
    setIsGenerating(true);
    try {
      const res = await fetch('http://localhost:8080/api/generate_image', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt })
      });
      const data = await res.json();
      if (data.url) {
        onGenerate('http://localhost:8080' + data.url);
      }
    } catch (err) {
      console.error(err);
    }
    setIsGenerating(false);
  };

  return (
    <div className="w-full h-full flex flex-col">
      <div className="h-14 flex items-center justify-between px-5 border-b border-white/5 shrink-0">
        <span className="text-xs font-semibold uppercase tracking-wider text-text-muted">Generation Engine</span>
        <button onClick={onClose} className="text-text-muted hover:text-white transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-5 space-y-8">
        <div className="space-y-3">
          <label className="text-[10px] uppercase tracking-widest text-text-muted font-semibold block">Prompt</label>
          <div className="relative group">
             <div className="absolute -inset-0.5 bg-gradient-to-r from-fuchsia-500 to-violet-500 rounded-xl blur opacity-20 group-hover:opacity-40 transition duration-500"></div>
             <textarea 
               value={prompt}
               onChange={e => setPrompt(e.target.value)}
               className="relative w-full h-32 bg-[#0a0a0c] border border-white/10 rounded-xl p-4 text-sm text-text-primary placeholder:text-text-muted/50 focus:outline-none focus:border-fuchsia-500/50 resize-none"
               placeholder="Describe the image you want to generate..."
             />
          </div>
        </div>

        <button 
          onClick={handleGenerate}
          disabled={isGenerating || !prompt.trim()}
          className="w-full relative group overflow-hidden rounded-xl p-[1px] disabled:opacity-50"
        >
          <span className="absolute inset-0 bg-gradient-to-r from-fuchsia-500 to-violet-500" />
          <div className="relative bg-[#0a0a0c] px-4 py-3 rounded-xl transition-all group-hover:bg-opacity-0 flex items-center justify-center gap-2">
            {isGenerating ? (
              <span className="text-sm font-semibold text-white">Generating...</span>
            ) : (
              <>
                <svg className="w-4 h-4 text-fuchsia-400 group-hover:text-white transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
                <span className="text-sm font-semibold text-white">Generate Asset</span>
              </>
            )}
          </div>
        </button>
      </div>
    </div>
  );
}
