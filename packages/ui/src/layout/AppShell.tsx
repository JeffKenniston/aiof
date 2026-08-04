import React, { useEffect } from 'react';
import { useLayoutStore } from '@aiof/rxdb-store';
import { motion, AnimatePresence } from 'framer-motion';
import { PanelLeft, PanelRight, PanelBottom, LayoutDashboard, MessageSquare, Terminal, Settings } from 'lucide-react';

export default function AppShell({ children }: { children: React.ReactNode }) {
  const {
    sidebarCollapsed, toggleSidebar,
    rightSidebarCollapsed, toggleRightSidebar,
    footerCollapsed, toggleFooter,
  } = useLayoutStore();

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
      const cmdOrCtrl = isMac ? e.metaKey : e.ctrlKey;

      if (cmdOrCtrl && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        toggleSidebar();
      } else if (cmdOrCtrl && e.key.toLowerCase() === 'j') {
        e.preventDefault();
        toggleFooter();
      } else if (cmdOrCtrl && e.key.toLowerCase() === '\\') {
        e.preventDefault();
        toggleRightSidebar();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [toggleSidebar, toggleFooter, toggleRightSidebar]);

  return (
    <div className="flex flex-col h-screen w-full bg-surface-base text-text-primary font-sans overflow-hidden">
      
      {/* THIN HEADER */}
      <header className="h-10 shrink-0 border-b border-border-default bg-surface-raised flex items-center justify-between px-3">
        <div className="flex items-center gap-3">
          <button onClick={toggleSidebar} className="p-1 hover:bg-surface-overlay rounded text-text-muted hover:text-text-primary transition-colors">
            <PanelLeft size={16} />
          </button>
          <div className="flex items-center gap-2 text-sm font-medium">
            <span className="text-accent-blue font-bold tracking-tight">Antigravity Workspace</span>
            <span className="text-text-muted text-xs border border-border-subtle px-1.5 py-0.5 rounded-sm bg-surface-overlay">v0.2.0-dev</span>
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs text-text-muted bg-surface-overlay px-2 py-1 rounded-md border border-border-subtle">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
            Daemon Connected
          </div>
          <button onClick={toggleRightSidebar} className="p-1 hover:bg-surface-overlay rounded text-text-muted hover:text-text-primary transition-colors">
            <PanelRight size={16} />
          </button>
        </div>
      </header>

      {/* MAIN CONTENT ROW */}
      <div className="flex flex-1 overflow-hidden relative">
        
        {/* LEFT NAV BAR (Slideable) */}
        <AnimatePresence initial={false}>
          {!sidebarCollapsed && (
            <motion.aside
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 260, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
              className="h-full border-r border-border-default bg-surface-raised shrink-0 flex flex-col overflow-hidden"
            >
              <div className="p-3 text-xs font-semibold text-text-muted uppercase tracking-wider">
                Workspaces
              </div>
              <nav className="flex-1 px-2 space-y-1">
                <button className="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded bg-surface-overlay text-text-primary font-medium">
                  <LayoutDashboard size={16} className="text-accent-blue" />
                  General
                </button>
                <button className="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded hover:bg-surface-overlay text-text-muted transition-colors">
                  <MessageSquare size={16} />
                  Chat Threads
                </button>
                <button className="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded hover:bg-surface-overlay text-text-muted transition-colors">
                  <Settings size={16} />
                  Settings
                </button>
              </nav>
            </motion.aside>
          )}
        </AnimatePresence>

        {/* CENTRAL WORKSPACE (Chat) */}
        <main className="flex-1 flex flex-col min-w-0 bg-surface-base relative overflow-hidden">
          {children}
        </main>

        {/* RIGHT NAV BAR (Slideable) */}
        <AnimatePresence initial={false}>
          {!rightSidebarCollapsed && (
            <motion.aside
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 320, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
              className="h-full border-l border-border-default bg-surface-raised shrink-0 flex flex-col overflow-hidden"
            >
              <div className="p-3 border-b border-border-default flex items-center justify-between">
                <span className="text-sm font-semibold text-text-primary">Artifact Canvas</span>
                <button onClick={toggleRightSidebar} className="p-1 hover:bg-surface-overlay rounded text-text-muted transition-colors">
                  <PanelRight size={14} />
                </button>
              </div>
              <div className="flex-1 p-4 text-sm text-text-muted flex flex-col items-center justify-center text-center">
                <div className="w-16 h-16 rounded-full border border-dashed border-border-subtle flex items-center justify-center mb-3">
                  <span className="text-2xl opacity-50">🎨</span>
                </div>
                <p>Waiting for Canvas Events...</p>
                <p className="text-xs opacity-70 mt-1">Multi-modal artifacts will render here.</p>
              </div>
            </motion.aside>
          )}
        </AnimatePresence>

      </div>

      {/* THINNER FOOTER (Slideable) */}
      <AnimatePresence initial={false}>
        {!footerCollapsed && (
          <motion.footer
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 180, opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ type: 'spring', bounce: 0, duration: 0.3 }}
            className="w-full border-t border-border-default bg-[#1e1e1e] shrink-0 flex flex-col overflow-hidden text-gray-300"
          >
            <div className="h-8 border-b border-white/10 flex items-center justify-between px-3 bg-[#252526]">
              <div className="flex items-center gap-4 text-xs font-medium">
                <button className="hover:text-white border-b-2 border-accent-blue px-1 py-1.5 text-white">Terminal</button>
                <button className="hover:text-white border-b-2 border-transparent px-1 py-1.5">eBPF Telemetry</button>
                <button className="hover:text-white border-b-2 border-transparent px-1 py-1.5">System Metrics</button>
              </div>
              <button onClick={toggleFooter} className="p-1 hover:bg-white/10 rounded text-gray-400 transition-colors">
                <PanelBottom size={14} />
              </button>
            </div>
            <div className="flex-1 p-2 font-mono text-[13px] overflow-y-auto">
              <div className="flex items-center gap-2 text-emerald-400">
                <Terminal size={14} />
                <span>root@aiof-workspace:~# </span>
              </div>
            </div>
          </motion.footer>
        )}
      </AnimatePresence>

      {/* Toggle Footer button when closed (shows as a tiny sliver at the absolute bottom right) */}
      <AnimatePresence>
        {footerCollapsed && (
          <motion.div 
            initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 10 }}
            className="absolute bottom-2 right-4 z-50"
          >
            <button 
              onClick={toggleFooter} 
              className="p-1.5 bg-surface-raised border border-border-default rounded-full shadow-lg text-text-muted hover:text-text-primary hover:bg-surface-overlay transition-colors"
              title="Toggle Terminal/Telemetry (Ctrl+J)"
            >
              <PanelBottom size={14} />
            </button>
          </motion.div>
        )}
      </AnimatePresence>

    </div>
  );
}
