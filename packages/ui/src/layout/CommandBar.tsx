import { useLayoutStore } from '@aiof/rxdb-store';
import React, { useState, useEffect } from 'react';
import {} from '@aiof/rxdb-store';
import { PRESETS } from '../workspace/presets';
import { BACKEND_URL } from '../config';

interface CommandBarProps {
  currentPreset: string;
  onPresetChange: (name: string) => void;
}

export default function CommandBar({ currentPreset, onPresetChange }: CommandBarProps) {
  const { activeProject, viewModeOverride, setViewModeOverride } = useLayoutStore();
  const [isConnected, setIsConnected] = useState(true);

  const searchInputRef = React.useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleFocusSearch = () => {
      searchInputRef.current?.focus();
      // Optional: select all text in the input
      searchInputRef.current?.select();
    };
    window.addEventListener('focus-search', handleFocusSearch);
    return () => window.removeEventListener('focus-search', handleFocusSearch);
  }, []);

  useEffect(() => {
    fetch(`${BACKEND_URL}/api/projects`)
      .then(() => {
        setIsConnected(true);
      })
      .catch(() => setIsConnected(false));
  }, []);


  return (
    <div className="h-12 bg-surface-glass backdrop-blur-md border-b border-border-default flex items-center justify-between px-4 z-commandbar shrink-0">
      
      <div className="flex items-center gap-4 flex-1">
        <button 
          onClick={() => useLayoutStore.getState().setProjectManagerOpen(true)}
          className="bg-surface-raised border border-border-default hover:border-accent-blue text-text-primary text-sm rounded-md px-4 py-1.5 focus:outline-none transition-colors flex items-center gap-2 group"
          title="Switch Project (Ctrl+Shift+P)"
        >
          <svg className="w-4 h-4 text-accent-blue group-hover:scale-110 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          {activeProject ? activeProject.split('/').pop() : 'Select Project...'}
        </button>
      </div>
      
      <div className="flex-2 flex justify-start hidden md:flex pl-4">
        <div className="relative w-64 max-w-md">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <svg className="h-4 w-4 text-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <input 
            ref={searchInputRef}
            type="text" 
            className="w-full bg-surface-raised border border-border-default text-text-primary text-sm rounded-full pl-10 pr-4 py-1.5 focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue" 
            placeholder="Search files, commands (Ctrl+P)..." 
          />
        </div>
      </div>
      
      <div className="flex items-center gap-3 flex-1 justify-end">
        {/* View Mode Override Selector */}
        <div className="flex items-center gap-1.5 hidden sm:flex">
          <select 
            className="bg-surface-raised border border-border-default hover:border-accent-blue text-text-primary text-xs rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-1 focus:ring-accent-blue cursor-pointer transition-colors"
            value={viewModeOverride}
            onChange={(e) => setViewModeOverride(e.target.value as any)}
            title="Force View Mode (Responsive / Phone / Tablet / Desktop / Workstation)"
          >
            <option value="auto">🤖 Auto View</option>
            <option value="condensed">📱 Condensed</option>
            <option value="semi-full">📟 Semi-Full</option>
            <option value="full">💻 Full</option>
            <option value="expanded">🖥️ Expanded</option>
          </select>
        </div>

        <div className="w-px h-5 bg-border-default mx-1 hidden sm:block"></div>

        {/* Workspace Preset Selector */}
        <div className="flex items-center gap-1.5 hidden sm:flex">
          <select 
            className="bg-surface-raised border border-border-default hover:border-accent-blue text-text-primary text-xs rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-1 focus:ring-accent-blue cursor-pointer transition-colors"
            value={currentPreset}
            onChange={(e) => onPresetChange(e.target.value)}
            title="Switch Workspace"
          >
            {PRESETS.map(p => (
              <option key={p.id} value={p.id}>{p.icon} {p.name}</option>
            ))}
          </select>
        </div>
        
        <div className="w-px h-5 bg-border-default mx-1 hidden sm:block"></div>
        
        <button className="text-text-muted hover:text-text-primary transition-colors relative">
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg>
          <span className="absolute top-0 right-0 block h-2 w-2 rounded-full bg-accent-blue ring-2 ring-surface-base"></span>
        </button>
        
        <div className={`w-2.5 h-2.5 rounded-full ${isConnected ? 'bg-accent-green' : 'bg-accent-red'} ml-2`} title={isConnected ? 'Backend Connected' : 'Backend Disconnected'}></div>
        
        <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-accent-blue to-accent-purple flex items-center justify-center text-white font-bold text-sm ml-2">
          J
        </div>
      </div>
      
    </div>
  );
}
