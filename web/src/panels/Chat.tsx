import { useState, useRef, useEffect } from 'react';

interface ChatMessage {
  id: string;
  role: 'user' | 'agent';
  content: string;
  agentName?: string;
}

import { pushHandler } from '../db/replication';
import { Document } from '../gen/proto/orchestrator/v1/orchestrator_pb';
import { useLayoutStore } from '../hooks/useLayoutStore';
import { getDatabase } from '../db';

interface ChatProps {
  hideHeader?: boolean;
  transparentBg?: boolean;
  panelId?: string;
}

export default function Chat(props: ChatProps) {
  const { hideHeader = false, transparentBg = false } = props;
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [model, setModel] = useState(() => localStorage.getItem('aiof-chat-model') || 'auto');
  const activeProject = useLayoutStore(state => state.activeProject);
  
  useEffect(() => {
    if (!activeProject) return;
    let sub: any;
    getDatabase(activeProject).then(db => {
      sub = db.chat_messages.find({ sort: [{ timestamp: 'asc' }] }).$.subscribe(docs => {
        setMessages(docs.map(d => ({
          id: d.id,
          role: d.role as 'user' | 'agent',
          content: d.content,
          agentName: d.agentName
        })));
      });
    });
    return () => {
      if (sub) sub.unsubscribe();
    };
  }, [activeProject]);

  useEffect(() => {
    localStorage.setItem('aiof-chat-model', model);
  }, [model]);
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
        timestamp: Date.now() 
    };
    
    setInput('');
    setIsTyping(true);

    try {
      const db = await getDatabase(activeProject);
      await db.chat_messages.insert(userMsg);

      const taskId = "task-" + Date.now();
      const viewMode = useLayoutStore.getState().viewModeOverride || "dev";
      const docPayload = JSON.stringify({
          id: taskId,
          task_id: taskId,
          column: "active",
          title: userMsg.content,
          project_id: activeProject,
          model_override: model === 'auto' ? undefined : model,
          view_mode: viewMode,
          created_at: Date.now(),
          agent: "Router",
          tier: "medium",
          priority: "medium",
          time: new Date().toLocaleTimeString()
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
          timestamp: Date.now()
      });
    } finally {
      setIsTyping(false);
    }
  };

  return (
    <div className={`w-full h-full flex flex-col text-sm ${transparentBg ? 'bg-transparent' : 'bg-surface-base'}`}>
      {!hideHeader && (
        <div className="h-10 bg-surface-raised border-b border-border-default flex items-center justify-between px-4 shrink-0">
          <div className="font-semibold text-text-primary flex items-center gap-2">
            <svg className="w-4 h-4 text-accent-blue" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" /></svg>
            Chat
          </div>
          <div className="flex items-center gap-2">
            <select 
              value={model}
              onChange={e => setModel(e.target.value)}
              className="bg-surface-base border border-border-default rounded px-2 py-1 text-xs text-text-secondary focus:outline-none focus:border-accent-blue cursor-pointer"
            >
              <option value="auto">Auto (Recommended)</option>
              <option value="gemini-3.1-pro-preview">Gemini 3.1 Pro</option>
              <option value="gemini-3.5-flash">Gemini 3.5 Flash</option>
            </select>
            <button 
              onClick={() => setMessages([{ id: Date.now().toString(), role: 'agent', agentName: 'Orchestrator', content: 'Chat history cleared. How can I help?' }])}
              className="text-xs text-text-muted hover:text-accent-red transition-colors px-2 py-1 rounded"
              title="Clear Chat"
            >
              Clear
            </button>
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
                <div className="text-[10px] font-bold text-accent-purple uppercase tracking-wider mb-1">
                  {msg.agentName}
                </div>
              )}
              <div className="whitespace-pre-wrap">{msg.content}</div>
            </div>
          </div>
        ))}
        {isTyping && (
          <div className="flex justify-start">
            <div className="bg-surface-raised border border-border-default rounded-2xl rounded-bl-none px-4 py-3 flex gap-1 items-center">
              <div className="w-1.5 h-1.5 bg-text-muted rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
              <div className="w-1.5 h-1.5 bg-text-muted rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
              <div className="w-1.5 h-1.5 bg-text-muted rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
            </div>
          </div>
        )}
      </div>

      <div className={`p-3 shrink-0 ${transparentBg ? 'bg-transparent' : 'bg-surface-raised border-t border-border-default'}`}>
        <form onSubmit={handleSend} className="flex gap-2 relative">
          <input 
            type="text" 
            className="flex-1 bg-surface-base border border-border-default rounded-full pl-4 pr-12 py-2.5 focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue text-text-primary transition-all"
            placeholder="Type a message..."
            value={input}
            onChange={e => setInput(e.target.value)}
          />
          <button 
            type="submit"
            disabled={!input.trim() || isTyping}
            className="absolute right-1 top-1 bottom-1 w-9 flex items-center justify-center bg-accent-blue text-white rounded-full hover:bg-blue-600 disabled:opacity-50 disabled:bg-surface-overlay transition-colors"
          >
            <svg className="w-4 h-4 translate-x-px" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
          </button>
        </form>
      </div>
    </div>
  );
}
