import { useState, Suspense } from 'react';
import { Bot, Save, Play, Plus, Trash2, Cpu, FileJson, TerminalSquare, Search } from 'lucide-react';
import { useLayoutStore } from '../hooks/useLayoutStore';

function FabricatorWorkstationContent() {
  
  

  const [agents, setAgents] = useState([
    { id: 'custom-1', name: 'Code Reviewer', prompt: 'You are a strict C++26 code reviewer...', model: 'gemini-2.5-pro' },
    { id: 'custom-2', name: 'SQL Optimizer', prompt: 'You optimize PostgreSQL queries...', model: 'gemini-2.5-flash' },
  ]);
  const [activeAgentId, setActiveAgentId] = useState('custom-1');
  const activeAgent = agents.find(a => a.id === activeAgentId);

  const handleSave = () => {
    // In a real implementation, we'd sync this to RxDB and the Go Backend catalog
    alert(`Saved ${activeAgent?.name} to dynamic catalog!`);
  };

  const handleUpdate = (updates: any) => {
    setAgents(prev => prev.map(a => a.id === activeAgentId ? { ...a, ...updates } : a));
  };

  const handleCreate = () => {
    const newId = crypto.randomUUID();
    setAgents([...agents, { id: newId, name: 'New Agent', prompt: 'You are a helpful assistant.', model: 'gemini-2.5-flash' }]);
    setActiveAgentId(newId);
  };

  return (
    <div className="flex h-full bg-neutral-950 text-neutral-50 overflow-hidden font-sans">
      {/* Sidebar List */}
      <div className="w-72 bg-neutral-900 border-r border-white/5 flex flex-col shrink-0">
        <div className="p-4 border-b border-white/5 flex items-center justify-between bg-black/20">
          <div className="flex items-center gap-2">
            <Bot className="w-5 h-5 text-accent-purple" />
            <h2 className="font-semibold text-sm tracking-wide uppercase">Agent Catalog</h2>
          </div>
          <button onClick={handleCreate} className="p-1 hover:bg-white/10 rounded transition-colors text-neutral-400 hover:text-white">
            <Plus className="w-4 h-4" />
          </button>
        </div>
        <div className="p-3">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-neutral-500" />
            <input 
              placeholder="Search catalog..." 
              className="w-full bg-black/40 border border-white/10 rounded-lg pl-9 pr-3 py-2 text-sm focus:outline-none focus:border-accent-purple/50 transition-colors"
            />
          </div>
        </div>
        <div className="flex-1 overflow-y-auto p-3 space-y-1">
          {agents.map(agent => (
            <button
              key={agent.id}
              onClick={() => setActiveAgentId(agent.id)}
              className={`w-full flex items-center gap-3 p-3 rounded-xl transition-all text-left ${activeAgentId === agent.id ? 'bg-accent-purple/10 border border-accent-purple/30 text-white shadow-lg' : 'bg-transparent border border-transparent text-neutral-400 hover:bg-white/5 hover:text-neutral-200'}`}
            >
              <div className={`w-8 h-8 rounded-lg flex items-center justify-center shrink-0 ${activeAgentId === agent.id ? 'bg-accent-purple text-white shadow-[0_0_15px_rgba(168,85,247,0.4)]' : 'bg-white/10'}`}>
                <Cpu className="w-4 h-4" />
              </div>
              <div className="truncate">
                <div className="font-medium text-sm truncate">{agent.name}</div>
                <div className="text-[10px] uppercase tracking-wider opacity-60 mt-0.5 font-mono">{agent.model}</div>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Main Fabricator Area */}
      {activeAgent ? (
        <div className="flex-1 flex flex-col min-w-0 bg-[#0A0A0A]">
          <header className="h-16 border-b border-white/5 flex items-center justify-between px-6 bg-black/20 shrink-0">
            <div className="flex items-center gap-4">
              <input 
                value={activeAgent.name}
                onChange={e => handleUpdate({ name: e.target.value })}
                className="bg-transparent border-none outline-none text-xl font-bold text-white placeholder-neutral-600 focus:ring-0 p-0"
                placeholder="Agent Name"
              />
              <span className="px-2.5 py-1 rounded bg-white/5 text-xs text-neutral-400 font-mono border border-white/5 flex items-center gap-1.5">
                <FileJson className="w-3.5 h-3.5" /> {activeAgent.id}
              </span>
            </div>
            <div className="flex items-center gap-3">
              <button onClick={() => setAgents(prev => prev.filter(a => a.id !== activeAgentId))} className="p-2 text-neutral-500 hover:text-red-400 hover:bg-red-400/10 rounded-lg transition-colors">
                <Trash2 className="w-4 h-4" />
              </button>
              <button className="flex items-center gap-2 px-4 py-2 bg-white/5 hover:bg-white/10 text-white rounded-lg text-sm font-medium transition-colors border border-white/5">
                <Play className="w-4 h-4 text-emerald-400" /> Test
              </button>
              <button onClick={handleSave} className="flex items-center gap-2 px-4 py-2 bg-accent-purple hover:bg-purple-500 text-white rounded-lg text-sm font-medium shadow-[0_0_20px_rgba(168,85,247,0.3)] hover:shadow-[0_0_30px_rgba(168,85,247,0.5)] transition-all">
                <Save className="w-4 h-4" /> Publish to Catalog
              </button>
            </div>
          </header>

          <main className="flex-1 overflow-y-auto p-6 space-y-6">
            <div className="space-y-2 max-w-4xl">
              <label className="text-xs font-semibold uppercase tracking-wider text-neutral-400 flex items-center gap-2">
                <TerminalSquare className="w-4 h-4" /> System Prompt
              </label>
              <textarea 
                value={activeAgent.prompt}
                onChange={e => handleUpdate({ prompt: e.target.value })}
                className="w-full h-96 bg-black/40 border border-white/10 rounded-xl p-4 text-sm font-mono leading-relaxed text-neutral-300 outline-none focus:border-accent-purple/50 focus:bg-black/60 transition-all shadow-inner custom-scrollbar"
                placeholder="Define the agent's behavior, instructions, and constraints..."
              />
            </div>

            <div className="grid grid-cols-2 gap-6 max-w-4xl">
              <div className="space-y-2">
                <label className="text-xs font-semibold uppercase tracking-wider text-neutral-400">Underlying Model</label>
                <select 
                  value={activeAgent.model}
                  onChange={e => handleUpdate({ model: e.target.value })}
                  className="w-full bg-black/40 border border-white/10 rounded-xl px-4 py-3 text-sm text-neutral-200 outline-none focus:border-accent-purple/50 appearance-none"
                >
                  <option value="gemini-2.5-pro">Gemini 2.5 Pro (Deep Reasoning)</option>
                  <option value="gemini-2.5-flash">Gemini 2.5 Flash (Fast Execution)</option>
                  <option value="openvino-genai">OpenVINO GenAI (Local Edge)</option>
                </select>
              </div>
              <div className="space-y-2">
                <label className="text-xs font-semibold uppercase tracking-wider text-neutral-400">Temperature</label>
                <div className="flex items-center gap-4 bg-black/40 border border-white/10 rounded-xl px-4 py-3">
                  <input type="range" min="0" max="1" step="0.1" className="flex-1 accent-accent-purple" defaultValue="0.2" />
                  <span className="text-sm font-mono text-neutral-400 w-8 text-right">0.2</span>
                </div>
              </div>
            </div>
            
            <div className="max-w-4xl space-y-2 pt-4">
               <label className="text-xs font-semibold uppercase tracking-wider text-neutral-400">Attached Tools</label>
               <div className="flex flex-wrap gap-2">
                 {['grep_search', 'run_command', 'write_file', 'replace_file_content', 'mcp_call'].map(t => (
                   <div key={t} className="px-3 py-1.5 rounded-lg border border-accent-purple/30 bg-accent-purple/10 text-accent-purple text-xs font-mono flex items-center gap-1.5 cursor-pointer hover:bg-accent-purple/20 transition-colors">
                     {t}
                   </div>
                 ))}
                 <button className="px-3 py-1.5 rounded-lg border border-dashed border-white/20 text-neutral-400 hover:text-white hover:border-white/40 text-xs flex items-center gap-1 transition-colors">
                   <Plus className="w-3 h-3" /> Add Tool
                 </button>
               </div>
            </div>
          </main>
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center text-neutral-500 bg-[#0A0A0A]">
          Select or create an agent to configure.
        </div>
      )}
    </div>
  );
}

export default function FabricatorWorkstation() {
  const activeProject = useLayoutStore((state) => state.activeProject);
  if (!activeProject) return <div className="p-4 text-neutral-400 flex items-center justify-center h-full">Please open a project to view the Agent Fabricator.</div>;

  return (
    <Suspense fallback={<div className="p-4 text-neutral-400">Loading Fabricator...</div>}>
      <FabricatorWorkstationContent />
    </Suspense>
  );
}
