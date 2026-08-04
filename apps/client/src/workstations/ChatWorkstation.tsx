import React from 'react';
import { Bot, Settings, Plus, Play } from 'lucide-react';
import Chat from '../panels/Chat';
import { useLayoutStore } from '../hooks/useLayoutStore';

export default function ChatWorkstation() {
  const activeProject = useLayoutStore(state => state.activeProject);

  if (!activeProject) {
    return (
      <div className="w-full h-full flex items-center justify-center text-neutral-500 bg-neutral-950">
        <div className="flex flex-col items-center gap-6">
          <div className="w-10 h-10 rounded-full border-2 border-accent-blue/40 border-t-transparent animate-spin"></div>
          <span className="text-sm font-medium tracking-wide">Initializing workspace…</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full bg-neutral-950 text-neutral-50 overflow-hidden">
      {/* Sidebar: Active Agents */}
      <div className="w-64 bg-neutral-900/50 border-r border-white/5 flex flex-col">
        <div className="p-4 border-b border-white/5 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bot className="w-5 h-5 text-accent-orange" />
            <h2 className="font-semibold text-sm tracking-wide">ACTIVE AGENTS</h2>
          </div>
          <button className="p-1 hover:bg-white/10 rounded-md transition-colors" title="Create Custom Agent">
            <Plus className="w-4 h-4" />
          </button>
        </div>
        
        <div className="p-3 flex-1 overflow-y-auto space-y-2">
          <AgentCard name="Orchestrator" status="idle" icon={<Bot className="w-4 h-4 text-accent-blue" />} />
          <AgentCard name="React Developer" status="working" icon={<Settings className="w-4 h-4 text-accent-green" />} />
          <AgentCard name="Data Analyst" status="idle" icon={<Play className="w-4 h-4 text-accent-purple" />} />
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col h-full bg-black/20">
        <Chat panelId="chat-workstation" hideHeader={true} transparentBg={true} />
      </div>
    </div>
  );
}

function AgentCard({ name, status, icon }: { name: string, status: string, icon: React.ReactNode }) {
  return (
    <div className="w-full flex items-center justify-between p-3 rounded-xl bg-white/5 border border-white/5 hover:bg-white/10 transition-colors cursor-pointer group">
      <div className="flex items-center gap-3">
        <div className="p-2 rounded-lg bg-black/40 border border-white/5 group-hover:border-white/10 transition-colors">
          {icon}
        </div>
        <div className="text-left">
          <div className="text-sm font-medium text-neutral-200">{name}</div>
          <div className="text-xs text-neutral-500 flex items-center gap-1">
            <span className={`w-1.5 h-1.5 rounded-full ${status === 'working' ? 'bg-green-500 animate-pulse' : 'bg-neutral-600'}`} />
            {status}
          </div>
        </div>
      </div>
    </div>
  );
}
