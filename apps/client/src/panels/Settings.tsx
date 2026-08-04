import { useState, useEffect } from 'react';
import { useLayoutStore } from '../hooks/useLayoutStore';
import { BACKEND_URL } from '../config';

export default function Settings({  }: { panelId: string }) {
  const [activeTab, setActiveTab] = useState<'global' | 'project'>('global');
  
  // Global State
  const [apiKey, setApiKey] = useState('');
  const [showKey, setShowKey] = useState(false);
  
  // Appearance (Global)
  const [themeMode, setThemeMode] = useState('dark');
  const [primaryColor, setPrimaryColor] = useState('#3b82f6');
  const [secondaryColor, setSecondaryColor] = useState('#a855f7');
  const [trimColor, setTrimColor] = useState('#1e293b'); // mapped to border/surface overlay
  
  // Project State
  const activeProject = useLayoutStore(state => state.activeProject);
  const [planningModel, setPlanningModel] = useState('gemini-3.1-pro-preview');
  const [executionModel, setExecutionModel] = useState('gemini-3.5-flash');
  const [dbString, setDbString] = useState('postgres://...');
  const [syncToDisk, setSyncToDisk] = useState(false);
  const [validatorEnabled, setValidatorEnabled] = useState(true);

  // Load from local storage
  useEffect(() => {
    setApiKey(localStorage.getItem('aiof_gemini_key') || '');
    setThemeMode(localStorage.getItem('aiof_theme') || 'dark');
    
    // Load custom colors if set
    const savedPrimary = localStorage.getItem('aiof_color_primary');
    if (savedPrimary) setPrimaryColor(savedPrimary);
    
    const savedSecondary = localStorage.getItem('aiof_color_secondary');
    if (savedSecondary) setSecondaryColor(savedSecondary);
    
    const savedTrim = localStorage.getItem('aiof_color_trim');
    if (savedTrim) setTrimColor(savedTrim);
    
    // Attempt to load project config from backend
    fetch(`${BACKEND_URL}/api/settings/load`)
      .then(res => {
        if (!res.ok) throw new Error('Not found');
        return res.json();
      })
      .then(data => {
        setPlanningModel(data.planningModel || 'gemini-3.1-pro-preview');
        setExecutionModel(data.executionModel || 'gemini-3.5-flash');
        setDbString(data.dbString || 'postgres://...');
        setSyncToDisk(data.syncToDisk !== false); // defaults to true if loading from file
        setValidatorEnabled(data.validatorEnabled !== false);
      })
      .catch(() => {
        // Fallback to local storage if no sync file
        setPlanningModel(localStorage.getItem(`aiof_plan_${activeProject}`) || 'gemini-3.1-pro-preview');
        setExecutionModel(localStorage.getItem(`aiof_exec_${activeProject}`) || 'gemini-3.5-flash');
        setDbString(localStorage.getItem(`aiof_db_${activeProject}`) || 'postgres://...');
        setSyncToDisk(localStorage.getItem(`aiof_sync_${activeProject}`) === 'true');
        setValidatorEnabled(localStorage.getItem(`aiof_val_${activeProject}`) !== 'false');
      });
  }, [activeProject]);

  // Apply CSS variables
  useEffect(() => {
    document.documentElement.style.setProperty('--color-accent-blue', primaryColor);
    document.documentElement.style.setProperty('--color-accent-purple', secondaryColor);
    document.documentElement.style.setProperty('--color-border-default', trimColor);
  }, [primaryColor, secondaryColor, trimColor]);

  const saveGlobalSettings = () => {
    localStorage.setItem('aiof_gemini_key', apiKey);
    localStorage.setItem('aiof_theme', themeMode);
    localStorage.setItem('aiof_color_primary', primaryColor);
    localStorage.setItem('aiof_color_secondary', secondaryColor);
    localStorage.setItem('aiof_color_trim', trimColor);
    alert('Global settings saved!');
  };

  const saveProjectSettings = async () => {
    localStorage.setItem(`aiof_plan_${activeProject}`, planningModel);
    localStorage.setItem(`aiof_exec_${activeProject}`, executionModel);
    localStorage.setItem(`aiof_db_${activeProject}`, dbString);
    localStorage.setItem(`aiof_sync_${activeProject}`, syncToDisk.toString());
    localStorage.setItem(`aiof_val_${activeProject}`, validatorEnabled.toString());
    
    if (syncToDisk) {
      try {
        const res = await fetch(`${BACKEND_URL}/api/settings/sync`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            planningModel,
            executionModel,
            dbString,
            syncToDisk,
            validatorEnabled
          })
        });
        if (!res.ok) throw new Error('Failed to sync settings to disk');
      } catch (err) {
        console.error('Failed to sync settings to disk', err);
        alert('Settings saved locally, but failed to sync to codebase root.');
        return;
      }
    }
    
    alert('Project settings saved!');
  };

  return (
    <div className="w-full h-full bg-surface-base flex flex-col text-sm text-text-primary overflow-hidden">
      
      {/* Tab Header */}
      <div className="flex border-b border-border-default px-4 pt-2 shrink-0">
        <button 
          onClick={() => setActiveTab('global')}
          className={`px-4 py-2 font-medium border-b-2 transition-colors ${activeTab === 'global' ? 'border-accent-blue text-accent-blue' : 'border-transparent text-text-secondary hover:text-text-primary'}`}
        >
          Global Settings
        </button>
        <button 
          onClick={() => setActiveTab('project')}
          className={`px-4 py-2 font-medium border-b-2 transition-colors ${activeTab === 'project' ? 'border-accent-blue text-accent-blue' : 'border-transparent text-text-secondary hover:text-text-primary'}`}
        >
          Project Settings
        </button>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-2xl mx-auto space-y-8 pb-10">
          
          {activeTab === 'global' ? (
            <div className="space-y-8 animate-in fade-in slide-in-from-bottom-2 duration-300">
              {/* API Keys */}
              <div>
                <h2 className="text-lg font-semibold border-b border-border-default pb-2 mb-4 text-text-primary">Authentication & APIs</h2>
                <div className="space-y-4">
                  <div>
                    <label className="block text-text-secondary mb-1">Gemini API Key</label>
                    <div className="flex gap-2">
                      <input 
                        type={showKey ? "text" : "password"} 
                        className="flex-1 bg-surface-raised border border-border-default rounded px-3 py-2 focus:border-accent-blue focus:outline-none"
                        value={apiKey}
                        onChange={e => setApiKey(e.target.value)}
                        placeholder="AIzaSy..."
                      />
                      <button 
                        onClick={() => setShowKey(!showKey)}
                        className="px-3 py-2 bg-surface-overlay border border-border-default rounded hover:bg-surface-raised transition-colors text-text-muted"
                      >
                        {showKey ? 'Hide' : 'Show'}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              {/* Appearance */}
              <div>
                <h2 className="text-lg font-semibold border-b border-border-default pb-2 mb-4 text-text-primary">Appearance</h2>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-text-secondary mb-1">Theme Mode</label>
                    <select 
                      className="w-full bg-surface-raised border border-border-default rounded px-3 py-2 focus:border-accent-blue focus:outline-none"
                      value={themeMode}
                      onChange={e => setThemeMode(e.target.value)}
                    >
                      <option value="dark">Dark</option>
                      <option value="light">Light</option>
                      <option value="system">System</option>
                    </select>
                  </div>
                </div>
                
                <h3 className="text-sm font-medium text-text-secondary mt-6 mb-3">Theme Colors</h3>
                <div className="grid grid-cols-3 gap-4">
                  <div className="flex flex-col gap-2">
                    <label className="text-xs text-text-muted">Primary</label>
                    <div className="flex gap-2 items-center">
                      <input type="color" className="w-8 h-8 rounded cursor-pointer bg-transparent border-0 p-0" value={primaryColor} onChange={e => setPrimaryColor(e.target.value)} />
                      <span className="font-mono text-xs text-text-secondary">{primaryColor}</span>
                    </div>
                  </div>
                  <div className="flex flex-col gap-2">
                    <label className="text-xs text-text-muted">Secondary</label>
                    <div className="flex gap-2 items-center">
                      <input type="color" className="w-8 h-8 rounded cursor-pointer bg-transparent border-0 p-0" value={secondaryColor} onChange={e => setSecondaryColor(e.target.value)} />
                      <span className="font-mono text-xs text-text-secondary">{secondaryColor}</span>
                    </div>
                  </div>
                  <div className="flex flex-col gap-2">
                    <label className="text-xs text-text-muted">Trim</label>
                    <div className="flex gap-2 items-center">
                      <input type="color" className="w-8 h-8 rounded cursor-pointer bg-transparent border-0 p-0" value={trimColor} onChange={e => setTrimColor(e.target.value)} />
                      <span className="font-mono text-xs text-text-secondary">{trimColor}</span>
                    </div>
                  </div>
                </div>
              </div>
              
              <div className="pt-4 flex justify-end border-t border-border-default">
                <button 
                  onClick={saveGlobalSettings}
                  className="px-6 py-2 bg-accent-blue text-white font-medium rounded hover:bg-blue-600 transition-colors shadow-lg shadow-blue-500/20"
                >
                  Save Global Settings
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-8 animate-in fade-in slide-in-from-bottom-2 duration-300">
              
              {/* Active Project Banner */}
              <div className="bg-surface-raised border border-border-active/30 rounded p-4 flex items-center justify-between">
                <div>
                  <div className="text-xs text-text-muted mb-1">Active Project Context</div>
                  <div className="font-mono text-sm text-text-primary break-all">{activeProject || 'No project loaded'}</div>
                </div>
                <div className="w-2 h-2 rounded-full bg-accent-green animate-pulse shrink-0 ml-4"></div>
              </div>

              {/* Orchestration Models */}
              <div>
                <h2 className="text-lg font-semibold border-b border-border-default pb-2 mb-4 text-text-primary">Orchestration Models</h2>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-text-secondary mb-1">Planning Model</label>
                    <select 
                      className="w-full bg-surface-raised border border-border-default rounded px-3 py-2 focus:border-accent-blue focus:outline-none"
                      value={planningModel}
                      onChange={e => setPlanningModel(e.target.value)}
                    >
                      <option value="gemini-3.1-pro-preview">Gemini 3.1 Pro (Preview)</option>
                      <option value="gemini-1.5-pro">Gemini 1.5 Pro</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-text-secondary mb-1">Execution Model</label>
                    <select 
                      className="w-full bg-surface-raised border border-border-default rounded px-3 py-2 focus:border-accent-blue focus:outline-none"
                      value={executionModel}
                      onChange={e => setExecutionModel(e.target.value)}
                    >
                      <option value="gemini-3.5-flash">Gemini 3.5 Flash</option>
                      <option value="gemini-1.5-flash">Gemini 1.5 Flash</option>
                    </select>
                  </div>
                </div>
              </div>

              {/* Infrastructure */}
              <div>
                <h2 className="text-lg font-semibold border-b border-border-default pb-2 mb-4 text-text-primary">Infrastructure Context</h2>
                <div>
                  <label className="block text-text-secondary mb-1">PostgreSQL Connection String</label>
                  <input 
                    type="text" 
                    className="w-full bg-surface-raised border border-border-default rounded px-3 py-2 focus:border-accent-blue focus:outline-none"
                    value={dbString}
                    onChange={e => setDbString(e.target.value)}
                  />
                </div>
              </div>

              {/* Agent Environment */}
              <div>
                <h2 className="text-lg font-semibold border-b border-border-default pb-2 mb-4 text-text-primary">Agent Environment</h2>
                <div className="space-y-3">
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input 
                      type="checkbox" 
                      className="w-4 h-4 rounded border-border-default text-accent-blue focus:ring-accent-blue"
                      checked={validatorEnabled}
                      onChange={e => setValidatorEnabled(e.target.checked)}
                    />
                    <div>
                      <div className="text-text-primary">Enable ValidatorAgent</div>
                      <div className="text-xs text-text-muted">Allows the system to spawn an independent validator agent for post-execution reviews.</div>
                    </div>
                  </label>
                </div>
              </div>

              {/* Sync Config */}
              <div>
                <h2 className="text-lg font-semibold border-b border-border-default pb-2 mb-4 text-text-primary">Configuration Sync</h2>
                <div className="bg-surface-overlay border border-border-default rounded p-4">
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input 
                      type="checkbox" 
                      className="w-4 h-4 rounded border-border-default text-accent-blue focus:ring-accent-blue"
                      checked={syncToDisk}
                      onChange={e => setSyncToDisk(e.target.checked)}
                    />
                    <div>
                      <div className="text-text-primary font-medium">Persist to <code className="bg-surface-base px-1 py-0.5 rounded text-accent-amber">~/.aiof/config</code></div>
                      <div className="text-xs text-text-muted mt-1">If enabled, project settings and orchestration states will be persisted to your user home directory (~/.aiof), keeping your codebase repository clean.</div>
                    </div>
                  </label>
                </div>
              </div>
              
              <div className="pt-4 flex justify-end border-t border-border-default">
                <button 
                  onClick={saveProjectSettings}
                  className="px-6 py-2 bg-surface-raised border border-border-default text-text-primary font-medium rounded hover:bg-surface-overlay transition-colors"
                >
                  Save Project Settings
                </button>
              </div>

            </div>
          )}
          
        </div>
      </div>
    </div>
  );
}
