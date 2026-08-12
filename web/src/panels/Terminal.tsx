import { useEffect, useRef, useState } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { BACKEND_WS_URL } from '../config';
import { useLayoutStore } from '../hooks/useLayoutStore';

export default function Terminal({  }: { panelId: string }) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const termInstance = useRef<XTerm | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isDisposedRef = useRef(false);
  const activeProject = useLayoutStore(state => state.activeProject);
  
  const [isDisconnected, setIsDisconnected] = useState(false);

  const connectWs = () => {
    if (isDisposedRef.current) return;
    
    // Clear any existing timer
    if (reconnectTimerRef.current !== null) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }

    if (wsRef.current) {
        // Remove onclose handler temporarily so we don't trigger the reconnect logic again if we manually close it
        wsRef.current.onclose = null;
        wsRef.current.close();
    }
    
    const wsUrl = new URL(`${BACKEND_WS_URL}/api/terminal`);
    wsUrl.searchParams.set('cols', termInstance.current?.cols.toString() || '80');
    if (activeProject) {
        wsUrl.searchParams.set('project', activeProject);
    }
    const ws = new WebSocket(wsUrl.toString());
    wsRef.current = ws;

    ws.onopen = () => {
      setIsDisconnected(false);
      reconnectAttemptsRef.current = 0;
      termInstance.current?.writeln('\r\n\x1b[32mConnected.\x1b[0m\r\n');
    };

    ws.onmessage = async (e) => {
      if (e.data instanceof Blob) {
        termInstance.current?.write(await e.data.text());
      } else {
        termInstance.current?.write(e.data);
      }
    };

    ws.onclose = () => {
      if (isDisposedRef.current) return;
      setIsDisconnected(true);
      termInstance.current?.writeln('\r\n\x1b[31mDisconnected from orchestrator.\x1b[0m');
      
      // Auto-reconnect with exponential backoff
      reconnectAttemptsRef.current++;
      const timeout = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
      termInstance.current?.writeln(`\x1b[33mReconnecting in ${timeout / 1000}s...\x1b[0m`);
      
      reconnectTimerRef.current = window.setTimeout(connectWs, timeout);
    };

    termInstance.current?.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });
  };

  useEffect(() => {
    if (!terminalRef.current) return;

    const term = new XTerm({
      theme: {
        background: '#0a0e17',
        foreground: '#f1f5f9',
        cursor: '#3b82f6',
        cursorAccent: '#0a0e17',
        selectionBackground: 'rgba(59, 130, 246, 0.3)',
      },
      fontFamily: '"JetBrains Mono", "Fira Code", monospace',
      fontSize: 13,
      cursorBlink: true,
    });
    
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    fitAddon.fit();
    termInstance.current = term;

    term.writeln('\x1b[1;36mAIOF Terminal v1.0\x1b[0m — \x1b[32mConnecting to orchestrator...\x1b[0m');

    isDisposedRef.current = false;
    connectWs();

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);
    
    // Hack to fit after React layout settles
    setTimeout(() => fitAddon.fit(), 100);

    return () => {
      isDisposedRef.current = true;
      if (reconnectTimerRef.current !== null) {
        clearTimeout(reconnectTimerRef.current);
      }
      window.removeEventListener('resize', handleResize);
      if (wsRef.current) wsRef.current.close();
      term.dispose();
    };
  }, []);

  const handleManualReconnect = () => {
    termInstance.current?.writeln('\r\n\x1b[36mManual reconnect triggered...\x1b[0m');
    reconnectAttemptsRef.current = 0;
    connectWs();
  };

  return (
    <div className="w-full h-full relative">
      <div className="w-full h-full bg-surface-base p-1 overflow-hidden" ref={terminalRef}></div>
      {isDisconnected && (
        <div className="absolute top-2 right-4 z-10">
          <button 
            onClick={handleManualReconnect}
            className="px-3 py-1.5 bg-surface-overlay hover:bg-surface-glass border border-border-default hover:border-accent-blue rounded text-xs font-medium text-text-primary shadow-elevated transition-colors flex items-center gap-2 cursor-pointer"
          >
            <div className="w-2 h-2 rounded-full bg-accent-amber animate-pulse"></div>
            Reconnect Now
          </button>
        </div>
      )}
    </div>
  );
}
