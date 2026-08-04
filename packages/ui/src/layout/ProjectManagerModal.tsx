import { useLayoutStore } from '@aiof/rxdb-store';
import React, { useState, useEffect } from 'react';
import {} from '@aiof/rxdb-store';
import { BACKEND_URL } from '../config';

interface ProjectMeta {
  name: string;
  path: string;
}

export default function ProjectManagerModal() {
  const { isProjectManagerOpen, setProjectManagerOpen, setActiveProject } = useLayoutStore();
  const [activeTab, setActiveTab] = useState<'recent' | 'open' | 'create'>('recent');
  const [recentProjects, setRecentProjects] = useState<ProjectMeta[]>([]);
  const [openPath, setOpenPath] = useState('');
  const [createName, setCreateName] = useState('');
  const [createParentPath, setCreateParentPath] = useState('~/Development/Projects');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isProjectManagerOpen) {
      fetchProjects();
    }
  }, [isProjectManagerOpen]);

  const fetchProjects = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/projects`);
      if (res.ok) {
        const data = await res.json();
        setRecentProjects(data.projects || []);
        // Auto-switch to Open/Create if no recents
        if ((!data.projects || data.projects.length === 0) && activeTab === 'recent') {
          setActiveTab('create');
        }
      }
    } catch (err) {
      console.error('Failed to fetch projects', err);
    }
  };


  const handleSelectRecent = async (path: string) => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${BACKEND_URL}/api/projects/load`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path }),
      });
      if (res.ok) {
        setActiveProject(path);
        setProjectManagerOpen(false);
      } else {
        setError(await res.text());
      }
    } catch (err) {
      setError('Failed to load project.');
    } finally {
      setLoading(false);
    }
  };

  const handleOpenExisting = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!openPath) return;
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${BACKEND_URL}/api/projects/load`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: openPath }),
      });
      if (res.ok) {
        setActiveProject(openPath);
        setProjectManagerOpen(false);
        setOpenPath('');
      } else {
        setError(await res.text());
      }
    } catch (err) {
      setError('Failed to load project.');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateNew = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!createName || !createParentPath) return;
    setLoading(true);
    setError('');
    
    // We should send the absolute path to the backend. If they used ~, we let the backend handle it or it fails.
    try {
      const res = await fetch(`${BACKEND_URL}/api/projects/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: createName, parentPath: createParentPath }),
      });
      if (res.ok) {
        // We will just let the backend set activeProjectDir, and then we set it locally
        // But what is the full path? The backend knows, but we have to guess it.
        // It's better if the backend returned the activeProject in response, but for now we guess:
        const finalPath = createParentPath.endsWith('/') ? `${createParentPath}${createName}` : `${createParentPath}/${createName}`;
        setActiveProject(finalPath);
        setProjectManagerOpen(false);
        setCreateName('');
      } else {
        setError(await res.text());
      }
    } catch (err) {
      setError('Failed to create project.');
    } finally {
      setLoading(false);
    }
  };

  if (!isProjectManagerOpen) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
      <div className="bg-surface-base border border-border-default rounded-xl shadow-2xl w-full max-w-2xl overflow-hidden flex flex-col">
        
        {/* Header */}
        <div className="p-6 border-b border-border-default bg-surface-raised flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold text-text-primary">Project Manager</h2>
            <p className="text-sm text-text-muted mt-1">Select or create an AIOF workspace</p>
          </div>
          <button 
            onClick={() => setProjectManagerOpen(false)}
            className="p-2 text-text-muted hover:text-text-primary hover:bg-surface-glass rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Tabs */}
        <div className="flex px-6 border-b border-border-default bg-surface-raised">
          <button
            className={`py-3 px-4 text-sm font-medium border-b-2 transition-colors ${activeTab === 'recent' ? 'border-accent-blue text-accent-blue' : 'border-transparent text-text-muted hover:text-text-primary'}`}
            onClick={() => setActiveTab('recent')}
          >
            Recent Projects
          </button>
          <button
            className={`py-3 px-4 text-sm font-medium border-b-2 transition-colors ${activeTab === 'open' ? 'border-accent-blue text-accent-blue' : 'border-transparent text-text-muted hover:text-text-primary'}`}
            onClick={() => setActiveTab('open')}
          >
            Open Existing
          </button>
          <button
            className={`py-3 px-4 text-sm font-medium border-b-2 transition-colors ${activeTab === 'create' ? 'border-accent-blue text-accent-blue' : 'border-transparent text-text-muted hover:text-text-primary'}`}
            onClick={() => setActiveTab('create')}
          >
            Create New
          </button>
        </div>

        {/* Content */}
        <div className="p-6 bg-surface-base min-h-[300px]">
          {error && (
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          {activeTab === 'recent' && (
            <div className="space-y-2 max-h-[300px] overflow-y-auto pr-2 custom-scrollbar">
              {recentProjects.length === 0 ? (
                <div className="text-center py-10 text-text-muted">
                  No recent projects found.
                </div>
              ) : (
                recentProjects.map((p) => (
                  <button
                    key={p.path}
                    onClick={() => handleSelectRecent(p.path)}
                    disabled={loading}
                    className="w-full text-left p-4 rounded-lg bg-surface-raised hover:bg-surface-glass border border-transparent hover:border-border-default transition-all group flex items-center justify-between"
                  >
                    <div>
                      <h3 className="font-medium text-text-primary group-hover:text-accent-blue transition-colors">{p.name}</h3>
                      <p className="text-xs text-text-muted mt-1 font-mono break-all">{p.path}</p>
                    </div>
                    <svg className="w-5 h-5 text-text-muted group-hover:text-accent-blue opacity-0 group-hover:opacity-100 transition-all" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                  </button>
                ))
              )}
            </div>
          )}

          {activeTab === 'open' && (
            <form onSubmit={handleOpenExisting} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-text-primary mb-1">Absolute Directory Path</label>
                <input
                  type="text"
                  value={openPath}
                  onChange={(e) => setOpenPath(e.target.value)}
                  placeholder="/home/user/Development/my-project"
                  className="w-full bg-surface-raised border border-border-default text-text-primary text-sm rounded-lg px-4 py-2.5 focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue"
                  required
                />
              </div>
              <button
                type="submit"
                disabled={loading || !openPath}
                className="w-full bg-accent-blue hover:bg-blue-600 text-white font-medium rounded-lg px-4 py-2.5 transition-colors disabled:opacity-50"
              >
                {loading ? 'Loading...' : 'Open Project'}
              </button>
            </form>
          )}

          {activeTab === 'create' && (
            <form onSubmit={handleCreateNew} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-text-primary mb-1">Project Name</label>
                <input
                  type="text"
                  value={createName}
                  onChange={(e) => setCreateName(e.target.value)}
                  placeholder="my-awesome-agent"
                  className="w-full bg-surface-raised border border-border-default text-text-primary text-sm rounded-lg px-4 py-2.5 focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-primary mb-1">Location (Absolute Parent Directory)</label>
                <input
                  type="text"
                  value={createParentPath}
                  onChange={(e) => setCreateParentPath(e.target.value)}
                  placeholder="/home/user/Development/Projects"
                  className="w-full bg-surface-raised border border-border-default text-text-primary text-sm rounded-lg px-4 py-2.5 focus:outline-none focus:border-accent-blue focus:ring-1 focus:ring-accent-blue"
                  required
                />
              </div>
              <div className="pt-2">
                <button
                  type="submit"
                  disabled={loading || !createName || !createParentPath}
                  className="w-full bg-accent-green hover:bg-green-600 text-white font-medium rounded-lg px-4 py-2.5 transition-colors disabled:opacity-50"
                >
                  {loading ? 'Creating...' : 'Create Project'}
                </button>
              </div>
            </form>
          )}
        </div>
        
      </div>
    </div>
  );
}
