import { useState, useRef, useEffect } from 'react';
import { MessageSquarePlus, MoreVertical, Settings, X, MessageSquare, Check, Plus, Mic, Square, Code, Bug, Shield, Layout, Database, TerminalSquare, User, Pin } from 'lucide-react';
import { pushHandler } from '../db/replication';
import { Document } from '../gen/proto/orchestrator/v1/orchestrator_pb';
import { useLayoutStore } from '../hooks/useLayoutStore';
import { getDatabase } from '../db';

interface ChatMessage {
  id: string;
  role: 'user' | 'agent';
  content: string;
  agentName?: string;
  modelUsed?: string;
  reasoning?: string[];
}

interface ChatThread {
  id: string;
  title: string;
  updatedAt: number;
  pinned?: boolean;
}

interface ChatProps {
  hideHeader?: boolean;
  transparentBg?: boolean;
  panelId?: string;
}

export default function Chat(props: ChatProps) {
  const { hideHeader = false, transparentBg = false } = props;
  const [messages, setMessages] = useState<ChatMessage[]>([]);

  const [model, setModel] = useState(() => localStorage.getItem('aiof-chat-model') || 'auto');
  const [extendedThinking, setExtendedThinking] = useState(() => localStorage.getItem('aiof-chat-extended') === 'true');
  
  const activeProject = useLayoutStore(state => state.activeProject);
  const activeWorkspaceId = useLayoutStore(state => state.activeWorkspaceId);
  const setGlobalChatOpen = useLayoutStore(state => state.setGlobalChatOpen);

  const [threadId, setThreadId] = useState(() => {
    return localStorage.getItem('aiof-chat-thread-global') || Date.now().toString();
  });
  const [threads, setThreads] = useState<ChatThread[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [autoExecute, setAutoExecute] = useState(() => localStorage.getItem('aiof-chat-autoexec') === 'true');
  const [personality, setPersonality] = useState(() => localStorage.getItem('aiof-chat-personality') || 'professional');
  
  const [selectedAgent, setSelectedAgent] = useState('Router');
  const [showThinking, setShowThinking] = useState(true);
  const [activeTaskId, setActiveTaskId] = useState<string | null>(null);
  
  const agents = [
    { id: 'Router', icon: <img src="/archimedes.png?v=2" className="w-6 h-6 object-contain" />, title: 'Archimedes (Router)' },
    { id: 'code_developer_agent', icon: <Code size={20}/>, title: 'Developer' },
    { id: 'qa_tester_agent', icon: <Bug size={20}/>, title: 'QA Tester' },
    { id: 'secops_agent', icon: <Shield size={20}/>, title: 'Security' },
    { id: 'ui_designer_agent', icon: <Layout size={20}/>, title: 'UI Designer' },
    { id: 'database_agent', icon: <Database size={20}/>, title: 'Database' },
    { id: 'devops_agent', icon: <TerminalSquare size={20}/>, title: 'DevOps' },
    { id: 'research_agent', icon: <User size={20}/>, title: 'Researcher' }
  ];
  
  useEffect(() => {
    if (!activeProject) return;
    localStorage.setItem('aiof-chat-thread-global', threadId);
    
    let sub: any;
    getDatabase(activeProject).then(db => {
      // Find messages for current thread
      sub = db.chat_messages.find({ 
        selector: { workspaceId: 'global', threadId: threadId },
        sort: [{ timestamp: 'asc' }] 
      }).$.subscribe(docs => {
        setMessages(docs.map(d => ({
          id: d.id,
          role: d.role as 'user' | 'agent',
          content: d.content,
          agentName: d.agentName,
          modelUsed: d.modelUsed,
          reasoning: d.reasoning
        })));
      });
      
      // Update thread list
      db.chat_threads.find({ selector: { workspaceId: 'global' } }).$.subscribe(docs => {
         setThreads(docs.map(d => ({
            id: d.id,
            title: d.title,
            updatedAt: d.updatedAt,
            pinned: d.pinned
         })).sort((a, b) => {
            if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
            return b.updatedAt - a.updatedAt;
         }));
      });
    });
    return () => {
      if (sub) sub.unsubscribe();
    };
  }, [activeProject, threadId]);

  useEffect(() => {
    localStorage.setItem('aiof-chat-model', model);
    localStorage.setItem('aiof-chat-extended', extendedThinking.toString());
    localStorage.setItem('aiof-chat-autoexec', autoExecute.toString());
    localStorage.setItem('aiof-chat-personality', personality);
  }, [model, extendedThinking, autoExecute, personality]);



  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isTyping]);

  const handleSend = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!input.trim() || !activeProject) return;

    const userMsg = { 
        id: Date.now().toString(), 
        role: 'user', 
        content: input, 
        timestamp: Date.now(),
        workspaceId: 'global',
        threadId: threadId
    };
    
    setInput('');
    setIsTyping(true);

    try {
      const db = await getDatabase(activeProject);
      await db.chat_messages.insert(userMsg);

      if (messages.length === 0) {
        const fallbackTitle = input.substring(0, 30);
        await db.chat_threads.insert({
            id: threadId,
            workspaceId: 'global',
            title: fallbackTitle,
            updatedAt: Date.now(),
            pinned: false
        });

        fetch('http://localhost:8080/api/chat/title', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prompt: input })
        }).then(res => res.ok ? res.json() : null).then(async data => {
            if (data && data.title) {
                const threadDoc = await db.chat_threads.findOne({ selector: { id: threadId } }).exec();
                if (threadDoc) {
                   await threadDoc.incrementalPatch({ title: data.title });
                }
            }
        }).catch(err => console.error("Title error:", err));
      } else {
        const threadDoc = await db.chat_threads.findOne({ selector: { id: threadId } }).exec();
        if (threadDoc) {
           await threadDoc.incrementalPatch({ updatedAt: Date.now() });
        }
      }

      const taskId = "task-" + Date.now();
      setActiveTaskId(taskId);
      const viewMode = useLayoutStore.getState().viewModeOverride || "dev";
      const docPayload = JSON.stringify({
          id: taskId,
          task_id: taskId,
          column: "active",
          title: userMsg.content,
          project_id: activeProject,
          model_override: model === 'auto' ? undefined : model,
          extended_thinking: extendedThinking,
          view_mode: viewMode,
          created_at: Date.now(),
          agent: selectedAgent,
          tier: "medium",
          priority: "medium",
          time: new Date().toLocaleTimeString(),
          workspaceId: activeWorkspaceId,
          threadId: threadId
      });
      
      const doc = new Document({
          id: taskId,
          documentType: "task_queue",
          payload: new TextEncoder().encode(docPayload),
          updatedAt: BigInt(Date.now()),
          isDeleted: false
      });
      
      await pushHandler([doc]);
    } catch (err: any) {
      console.error('Chat error:', err);
      const db = await getDatabase(activeProject);
      await db.chat_messages.insert({
          id: Date.now().toString(),
          role: 'agent',
          agentName: 'System',
          content: 'Error: ' + err.message,
          timestamp: Date.now(),
          workspaceId: 'global',
          threadId: threadId
      });
      setIsTyping(false);
    }
  };

  useEffect(() => {
    if (messages.length > 0 && messages[messages.length - 1].role === 'agent') {
      setIsTyping(false);
    }
  }, [messages]);

  return (
    <div className={`w-full h-full flex flex-row text-sm ${transparentBg ? 'bg-transparent' : 'bg-surface-base'}`}>
      
      {/* Agent Sidebar */}
      {!hideHeader && (
        <div className="w-14 shrink-0 bg-surface-raised border-r border-border-default flex flex-col items-center py-3 overflow-y-auto gap-3">
          {agents.map(a => (
            <button
              key={a.id}
              onClick={() => setSelectedAgent(a.id)}
              className={`w-10 h-10 rounded-full flex items-center justify-center transition-all ${selectedAgent === a.id ? 'bg-accent-blue text-white shadow-elevated scale-110' : 'bg-surface-base text-text-muted hover:bg-surface-overlay hover:text-text-primary'}`}
              title={a.title}
            >
              {a.icon}
            </button>
          ))}
        </div>
      )}

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col min-w-0">
        {!hideHeader && (
          <div className="h-12 bg-surface-raised border-b border-border-default flex items-center justify-between px-4 shrink-0 relative">
          <div className="font-semibold text-text-primary flex items-center gap-2">
            <img src="/archimedes.png?v=2" alt="Archimedes" className="w-5 h-5 object-contain" />
            Archimedes
          </div>
          <div className="flex items-center gap-1 relative">
            <button 
              onClick={() => {
                 setThreadId(Date.now().toString());
                 setMessages([]);
                 setMenuOpen(false);
              }}
              className="p-1.5 text-text-muted hover:text-text-primary hover:bg-surface-overlay rounded transition-colors"
              title="New Chat"
            >
              <MessageSquarePlus size={18} />
            </button>
            <button 
              onClick={() => setMenuOpen(!menuOpen)}
              className={`p-1.5 rounded transition-colors ${menuOpen ? 'bg-surface-overlay text-text-primary' : 'text-text-muted hover:text-text-primary hover:bg-surface-overlay'}`}
              title="Chat Options"
            >
              <MoreVertical size={18} />
            </button>
            <button 
              onClick={() => setGlobalChatOpen(false)}
              className="p-1.5 text-text-muted hover:text-text-primary hover:bg-surface-overlay rounded transition-colors"
              title="Close"
            >
              <X size={18} />
            </button>
            
            {menuOpen && (
              <>
                <div className="fixed inset-0 z-40" onClick={() => setMenuOpen(false)} />
                <div className="absolute top-10 right-8 w-64 bg-surface-raised border border-border-default shadow-elevated rounded-lg py-2 z-50 flex flex-col max-h-[400px]">
                  <div className="px-3 pb-2 text-xs font-semibold tracking-wider text-text-muted uppercase">Recent Threads</div>
                  <div className="flex-1 overflow-y-auto px-2 space-y-1">
                    {threads.map(t => (
                      <button 
                        key={t.id}
                        onClick={() => { setThreadId(t.id); setMenuOpen(false); }}
                        onContextMenu={(e) => {
                           e.preventDefault();
                           e.stopPropagation();
                           window.dispatchEvent(new CustomEvent('open-context-menu', {
                              detail: {
                                x: e.clientX,
                                y: e.clientY,
                                items: [
                                  { label: t.pinned ? 'Unpin Thread' : 'Pin Thread', action: async () => {
                                      const db = await getDatabase(activeProject);
                                      const doc = await db.chat_threads.findOne({ selector: { id: t.id } }).exec();
                                      if (doc) await doc.incrementalPatch({ pinned: !t.pinned });
                                  }},
                                  { label: 'Rename Thread', action: async () => {
                                      const newName = prompt("Enter new thread name:", t.title);
                                      if (newName) {
                                          const db = await getDatabase(activeProject);
                                          const doc = await db.chat_threads.findOne({ selector: { id: t.id } }).exec();
                                          if (doc) await doc.incrementalPatch({ title: newName });
                                      }
                                  }},
                                  { divider: true },
                                  { label: 'Delete Thread', danger: true, action: async () => {
                                      if (confirm("Delete this thread forever?")) {
                                          const db = await getDatabase(activeProject);
                                          const doc = await db.chat_threads.findOne({ selector: { id: t.id } }).exec();
                                          if (doc) await doc.remove();
                                          const msgs = await db.chat_messages.find({ selector: { threadId: t.id } }).exec();
                                          for (const m of msgs) await m.remove();
                                          if (threadId === t.id) {
                                              setThreadId(Date.now().toString());
                                              setMessages([]);
                                          }
                                      }
                                  }}
                                ]
                              }
                           }));
                        }}
                        className={`w-full text-left px-3 py-2 text-sm rounded flex items-center gap-2 transition-colors ${threadId === t.id ? 'bg-accent-blue/10 text-accent-blue' : 'text-text-secondary hover:bg-surface-overlay hover:text-text-primary'}`}
                      >
                        {t.pinned ? <Pin size={14} className="shrink-0 text-accent-blue" /> : <MessageSquare size={14} className="shrink-0" />}
                        <span className="truncate">{t.title || 'New Thread'}</span>
                        {threadId === t.id && <Check size={14} className="ml-auto shrink-0" />}
                      </button>
                    ))}
                    {threads.length === 0 && (
                      <div className="text-xs text-text-muted px-3 py-2">No previous threads</div>
                    )}
                  </div>
                  <div className="h-px bg-border-default my-2 w-full" />
                  <button 
                    onClick={() => { setMenuOpen(false); setShowSettings(true); }}
                    className="w-full text-left px-5 py-2 text-sm text-text-secondary hover:bg-surface-overlay hover:text-text-primary flex items-center gap-3 transition-colors"
                  >
                    <Settings size={16} />
                    Agent Chat Settings
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Settings Modal Overlay */}
      {showSettings && (
        <div className="absolute inset-0 z-50 bg-black/40 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-surface-raised border border-border-default rounded-xl shadow-2xl w-full max-w-sm overflow-hidden flex flex-col">
            <div className="px-4 py-3 border-b border-border-default flex justify-between items-center bg-surface-base">
              <h3 className="font-semibold text-text-primary flex items-center gap-2"><Settings size={16}/> Agent Settings</h3>
              <button onClick={() => setShowSettings(false)} className="text-text-muted hover:text-text-primary"><X size={16}/></button>
            </div>
            <div className="p-4 space-y-4 overflow-y-auto max-h-[60vh]">
              <div className="space-y-1">
                <label className="text-xs font-semibold text-text-secondary uppercase tracking-wider">Default Model</label>
                <select value={model} onChange={e => setModel(e.target.value)} className="w-full bg-surface-base border border-border-default rounded p-2 text-sm text-text-primary outline-none focus:border-accent-blue">
                  <option value="auto">Auto-Select (Router)</option>
                  <option value="flash">Gemini 2.5 Flash</option>
                  <option value="pro">Gemini 2.5 Pro</option>
                </select>
              </div>
              <div className="space-y-1">
                <label className="text-xs font-semibold text-text-secondary uppercase tracking-wider">Agent Personality</label>
                <select value={personality} onChange={e => setPersonality(e.target.value)} className="w-full bg-surface-base border border-border-default rounded p-2 text-sm text-text-primary outline-none focus:border-accent-blue">
                  <option value="professional">Professional & Concise</option>
                  <option value="casual">Casual & Friendly</option>
                  <option value="detailed">Verbose & Detailed</option>
                </select>
              </div>
              <div className="flex items-center justify-between mt-2 pt-2 border-t border-border-default/50">
                <span className="text-sm text-text-primary">Extended Thinking (Pro only)</span>
                <input type="checkbox" checked={extendedThinking} onChange={e => setExtendedThinking(e.target.checked)} className="w-4 h-4 accent-accent-blue cursor-pointer" />
              </div>
              <div className="flex items-center justify-between pt-1">
                <span className="text-sm text-text-primary">Auto-Execute Safe Commands</span>
                <input type="checkbox" checked={autoExecute} onChange={e => setAutoExecute(e.target.checked)} className="w-4 h-4 accent-accent-blue cursor-pointer" />
              </div>
            </div>
            <div className="p-3 border-t border-border-default bg-surface-base flex justify-end">
              <button onClick={() => setShowSettings(false)} className="px-4 py-1.5 bg-accent-blue hover:bg-blue-600 text-white rounded text-sm font-medium transition-colors">Done</button>
            </div>
          </div>
        </div>
      )}


      <div className="flex-1 overflow-y-auto p-4 space-y-4" ref={scrollRef}>
        {messages.map((msg) => (
          <div 
            key={msg.id} 
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
            onContextMenu={(e) => {
              e.preventDefault();
              window.dispatchEvent(new CustomEvent('open-context-menu', {
                detail: {
                  x: e.clientX,
                  y: e.clientY,
                  items: [
                    { label: 'Copy Text', action: () => navigator.clipboard.writeText(msg.content) },
                    { divider: true },
                    { label: 'Delete Message', danger: true, action: () => setMessages(prev => prev.filter(m => m.id !== msg.id)) }
                  ]
                }
              }));
            }}
          >
            <div className={`max-w-[85%] rounded-2xl px-4 py-2 ${
              msg.role === 'user' 
                ? 'bg-accent-blue text-white rounded-br-none' 
                : 'bg-surface-raised border border-border-default text-text-primary rounded-bl-none'
            }`}>
              {msg.role === 'agent' && (
                <div className="flex items-center gap-2 mb-1">
                  <div className="text-[10px] font-bold text-accent-purple uppercase tracking-wider">
                    {msg.agentName}
                  </div>
                  {msg.modelUsed && (
                    <div className="text-[9px] text-text-muted/40 font-mono tracking-tighter" title="Model Routed">
                      {msg.modelUsed}
                    </div>
                  )}
                </div>
              )}
              {msg.role === 'agent' && msg.reasoning && msg.reasoning.length > 0 && (
                <details className="mb-2">
                  <summary className="text-[10px] text-text-muted hover:text-text-primary cursor-pointer select-none outline-none flex items-center gap-1 opacity-70 w-fit">
                    <Code size={10} /> Show Thinking Process
                  </summary>
                  <div className="mt-1 bg-surface-overlay/50 border border-border-default rounded p-2 text-[10px] font-mono text-text-secondary space-y-1">
                    {msg.reasoning.map((r, i) => (
                      <div key={i}>[*] {r}</div>
                    ))}
                  </div>
                </details>
              )}
              <div className="whitespace-pre-wrap">{msg.content}</div>
            </div>
          </div>
        ))}
        {isTyping && (
          <div className="flex flex-col gap-2">
            <details 
              open={showThinking} 
              onToggle={e => setShowThinking(e.currentTarget.open)}
              className="bg-surface-overlay border border-border-default rounded-lg px-3 py-2 text-xs text-text-secondary cursor-pointer"
            >
              <summary className="font-semibold select-none flex items-center outline-none">
                <span className="flex items-center gap-2">
                  <div className="flex gap-1">
                    <div className="w-1 h-1 bg-accent-blue rounded-full animate-ping" />
                  </div>
                  {selectedAgent === 'Router' ? 'Routing Request...' : `${selectedAgent} is thinking...`}
                </span>
              </summary>
              <div className="mt-2 pt-2 border-t border-border-default/50 font-mono text-[10px] whitespace-pre-wrap opacity-80">
                [*] Analyzing task context and view mode...
                [*] Constructing prompt template...
                [*] Awaiting model response...
              </div>
            </details>
          </div>
        )}
      </div>

      <div className={`p-3 shrink-0 ${transparentBg ? 'bg-transparent' : 'bg-surface-raised border-t border-border-default'}`}>
        <form onSubmit={handleSend} className="flex gap-2 relative items-center">
          <label className="p-2 cursor-pointer text-text-muted hover:text-text-primary transition-colors shrink-0">
            <Plus size={20} />
            <input type="file" className="hidden" accept="image/*,application/pdf,text/plain,text/markdown,text/csv,application/json" multiple />
          </label>
          <input 
            type="text" 
            className="flex-1 min-w-0 bg-surface-base border border-border-default rounded-full px-4 py-2.5 focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue text-text-primary transition-all"
            placeholder={`Message ${selectedAgent}...`}
            value={input}
            onChange={e => setInput(e.target.value)}
          />
          <button type="button" className="p-2 text-text-muted hover:text-text-primary transition-colors shrink-0">
            <Mic size={20} />
          </button>
          
          {isTyping ? (
            <button 
              type="button"
              onClick={async () => {
                setIsTyping(false);
                if (activeTaskId && activeProject) {
                  const doc = new Document({
                    id: activeTaskId,
                    documentType: "task_queue",
                    payload: new TextEncoder().encode(JSON.stringify({ id: activeTaskId, column: "cancelled" })),
                    updatedAt: BigInt(Date.now()),
                    isDeleted: false
                  });
                  await pushHandler([doc]);
                }
              }}
              className="w-10 h-10 flex items-center justify-center bg-accent-crimson text-white rounded-full hover:bg-red-600 transition-colors shrink-0"
              title="Stop Generation"
            >
              <Square size={16} fill="currentColor" />
            </button>
          ) : (
            <button 
              type="submit"
              disabled={!input.trim()}
              className="w-10 h-10 flex items-center justify-center bg-accent-blue text-white rounded-full hover:bg-blue-600 disabled:opacity-50 disabled:bg-surface-overlay transition-colors shrink-0"
            >
              <svg className="w-4 h-4 translate-x-px" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
            </button>
          )}
        </form>
      </div>
    </div>
    </div>
  );
}
