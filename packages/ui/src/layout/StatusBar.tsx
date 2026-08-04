import { useState, useEffect } from 'react';

export default function StatusBar() {
  const [time, setTime] = useState(new Date().toLocaleTimeString());
  const [cursor, setCursor] = useState({ line: 1, col: 1 });

  useEffect(() => {
    const timer = setInterval(() => {
      setTime(new Date().toLocaleTimeString());
    }, 1000);
    const handleCursor = (e: any) => {
      setCursor({ line: e.detail.line, col: e.detail.col });
    };
    window.addEventListener('editor-cursor', handleCursor);

    return () => {
      clearInterval(timer);
      window.removeEventListener('editor-cursor', handleCursor);
    };
  }, []);

  return (
    <div className="h-6 bg-surface-raised border-t border-border-default flex items-center justify-between px-4 z-commandbar shrink-0 hidden md:flex text-[11px] font-mono text-text-muted">
      
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1 hover:text-text-primary cursor-pointer transition-colors">
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2" /></svg>
          main
        </div>
        <div className="w-px h-3 bg-border-default"></div>
        <div className="hover:text-text-primary cursor-pointer transition-colors">Ln {cursor.line}, Col {cursor.col}</div>
      </div>
      
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-1.5" title="Go Backend Status">
          <div className="w-1.5 h-1.5 rounded-full bg-accent-green"></div>
          Go Backend: Connected
        </div>
        <div className="w-px h-3 bg-border-default"></div>
        <div className="flex items-center gap-1.5" title="RxDB Sync Status">
          <div className="w-1.5 h-1.5 rounded-full bg-accent-green"></div>
          RxDB: Synced
        </div>
      </div>
      
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1 hover:text-text-primary cursor-pointer transition-colors">
          <svg className="w-3.5 h-3.5 text-accent-red" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
          0
        </div>
        <div className="flex items-center gap-1 hover:text-text-primary cursor-pointer transition-colors">
          <svg className="w-3.5 h-3.5 text-accent-amber" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
          0
        </div>
        <div className="w-px h-3 bg-border-default"></div>
        <div>{time}</div>
      </div>
      
    </div>
  );
}
