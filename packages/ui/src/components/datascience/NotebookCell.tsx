import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

export default function NotebookCell({ type, content, output }: { type: 'markdown' | 'code' | 'ai', content: string, output?: any }) {
  const getThemeColors = () => {
    switch(type) {
      case 'code': return 'border-white/10 bg-[#0a0a0a]';
      case 'ai': return 'border-emerald-500/30 bg-emerald-500/5 shadow-[0_0_15px_rgba(16,185,129,0.05)]';
      default: return 'border-transparent bg-transparent';
    }
  };

  return (
    <div className="group relative">
      <div className="absolute -left-12 top-2 opacity-0 group-hover:opacity-100 transition-opacity">
        <button className="p-1.5 rounded bg-white/5 hover:bg-emerald-500/20 hover:text-emerald-400 text-text-muted transition-colors">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z" /></svg>
        </button>
      </div>

      <div className={`rounded-xl border ${getThemeColors()} transition-colors overflow-hidden`}>
        {type === 'ai' && (
          <div className="h-8 bg-emerald-500/10 border-b border-emerald-500/20 flex items-center px-4 gap-2">
            <span className="text-emerald-400">✨</span>
            <span className="text-xs font-medium text-emerald-400/80 uppercase tracking-wider">Agent Request</span>
          </div>
        )}
        
        <div className={`p-4 ${type === 'markdown' ? '' : 'pt-3'}`}>
          <pre className={`font-mono text-sm whitespace-pre-wrap ${type === 'markdown' ? 'font-sans text-text-primary text-base font-light' : 'text-text-secondary'}`}>
            {content}
          </pre>
        </div>

        {output && (
          <div className="border-t border-white/5 bg-white/[0.02] p-4">
            {output.type === 'chart' ? (
              <div className="h-64 rounded-lg border border-white/5 bg-[#0a0a0a] flex items-center justify-center text-text-muted text-sm shadow-inner relative overflow-hidden p-4">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={output.data}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#ffffff10" />
                    <XAxis dataKey="name" stroke="#ffffff50" />
                    <YAxis stroke="#ffffff50" />
                    <Tooltip contentStyle={{ backgroundColor: '#0a0a0a', borderColor: '#ffffff20' }} />
                    <Line type="monotone" dataKey="value" stroke="#10b981" />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <pre className="font-mono text-xs text-text-muted whitespace-pre-wrap">{output.data}</pre>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
