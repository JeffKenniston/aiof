import { useState, useEffect } from 'react';
import { BACKEND_URL } from '../config';

interface FileNode {
  name: string;
  type: 'file' | 'dir';
  size?: number;
  children?: FileNode[];
}

export default function FileExplorer({  }: { panelId: string }) {
  const [tree, setTree] = useState<FileNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({ 'src': true });
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [lastSelected, setLastSelected] = useState<string>('');

  useEffect(() => {
    const handleOpenFile = (e: any) => {
      setSelectedFiles(new Set([e.detail.filepath]));
      setLastSelected(e.detail.filepath);
    };
    const handleDbUpdate = (e: any) => {
      const { collection, doc } = e.detail;
      if (collection === 'task_queue' && doc.column === 'completed') {
        loadTree();
      }
    };
    
    window.addEventListener('open-file', handleOpenFile);
    window.addEventListener('db-update', handleDbUpdate);
    
    return () => {
      window.removeEventListener('open-file', handleOpenFile);
      window.removeEventListener('db-update', handleDbUpdate);
    };
  }, []);

  const [inlineEdit, setInlineEdit] = useState<{ path: string, type: 'rename' | 'new-file' | 'new-dir', initialValue: string } | null>(null);
  const [editValue, setEditValue] = useState('');

  const loadTree = () => {
    fetch(`${BACKEND_URL}/api/files`)
      .then(res => res.json())
      .then(data => {
        setTree(data);
        setLoading(false);
      })
      .catch((err) => {
        console.warn('Failed to fetch files:', err);
        setTree([]);
        setLoading(false);
      });
  };

  useEffect(() => {
    loadTree();
  }, []);

  const handleAction = async (action: 'rename' | 'delete' | 'create', oldPath: string, newPath?: string, isDir?: boolean) => {
    if (action === 'delete') {
      try {
        await fetch(`${BACKEND_URL}/api/delete`, {
          method: 'POST', body: JSON.stringify({ path: oldPath })
        });
      } catch (e) {}
    } else if (action === 'rename') {
      try {
        await fetch(`${BACKEND_URL}/api/rename`, {
          method: 'POST', body: JSON.stringify({ oldPath, newPath })
        });
      } catch (e) {}
    } else if (action === 'create') {
      try {
        await fetch(`${BACKEND_URL}/api/create_node`, {
          method: 'POST', body: JSON.stringify({ path: oldPath, isDir }) // oldPath used for the new full path here
        });
      } catch (e) {}
    }
    setInlineEdit(null);
    loadTree();
  };

  const toggleExpand = (path: string) => {
    setExpanded(prev => ({ ...prev, [path]: !prev[path] }));
  };

  const visibleNodes: { path: string }[] = [];
  const computeVisibleNodes = (nodes: FileNode[], pathPrefix = '') => {
    for (const node of nodes) {
      const fullPath = `${pathPrefix}/${node.name}`;
      const normalizedPath = fullPath.startsWith('/') ? fullPath.substring(1) : fullPath;
      visibleNodes.push({ path: normalizedPath });
      if (node.type === 'dir' && expanded[fullPath] && node.children) {
        computeVisibleNodes(node.children, fullPath);
      }
    }
  };
  computeVisibleNodes(tree);

  const handleNodeClick = (e: React.MouseEvent, normalizedPath: string, fullPath: string, isDir: boolean) => {
    e.stopPropagation();
    let newSelected = new Set(selectedFiles);

    if (e.shiftKey && lastSelected) {
      const lastIdx = visibleNodes.findIndex(n => n.path === lastSelected);
      const currIdx = visibleNodes.findIndex(n => n.path === normalizedPath);
      if (lastIdx !== -1 && currIdx !== -1) {
        const start = Math.min(lastIdx, currIdx);
        const end = Math.max(lastIdx, currIdx);
        if (!e.ctrlKey && !e.metaKey) {
          newSelected.clear();
        }
        for (let i = start; i <= end; i++) {
          newSelected.add(visibleNodes[i].path);
        }
      }
    } else if (e.ctrlKey || e.metaKey) {
      if (newSelected.has(normalizedPath)) {
        newSelected.delete(normalizedPath);
      } else {
        newSelected.add(normalizedPath);
      }
    } else {
      newSelected.clear();
      newSelected.add(normalizedPath);
      if (!isDir) {
        window.dispatchEvent(new CustomEvent('open-file', { detail: { filepath: normalizedPath } }));
      }
    }

    if (isDir && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
      toggleExpand(fullPath);
    }

    setSelectedFiles(newSelected);
    setLastSelected(normalizedPath);
  };

  const renderTree = (nodes: FileNode[], pathPrefix = '') => {
    return nodes.map(node => {
      const fullPath = `${pathPrefix}/${node.name}`;
      const isDir = node.type === 'dir';
      const isExpanded = !!expanded[fullPath];
      const normalizedPath = fullPath.startsWith('/') ? fullPath.substring(1) : fullPath;
      const isSelected = selectedFiles.has(normalizedPath);

      return (
        <div key={fullPath} className="text-sm font-mono text-text-secondary">
          <div 
            className={`flex items-center gap-1.5 py-1 px-2 cursor-pointer rounded ${isSelected ? 'bg-accent-blue/20 text-accent-blue' : 'hover:bg-surface-raised'}`}
            onClick={(e) => handleNodeClick(e, normalizedPath, fullPath, isDir)}
            onContextMenu={(e) => {
              e.preventDefault();
              e.stopPropagation();

              let contextSelected = new Set(selectedFiles);
              if (!contextSelected.has(normalizedPath)) {
                contextSelected.clear();
                contextSelected.add(normalizedPath);
                setSelectedFiles(contextSelected);
                setLastSelected(normalizedPath);
              }

              const multiCount = contextSelected.size;

              window.dispatchEvent(new CustomEvent('open-context-menu', {
                detail: {
                  x: e.clientX,
                  y: e.clientY,
                  items: multiCount > 1 ? [
                    { label: `Delete ${multiCount} Items`, danger: true, action: () => {
                      if(confirm(`Are you sure you want to delete ${multiCount} items?`)) {
                        contextSelected.forEach(p => handleAction('delete', p));
                      }
                    }},
                    { label: `Copy ${multiCount} Paths`, action: () => {
                      navigator.clipboard.writeText(Array.from(contextSelected).join('\n'));
                    }}
                  ] : [
                    { label: isDir ? 'New File' : 'Open', action: () => {
                      if (isDir) {
                        setExpanded(prev => ({ ...prev, [fullPath]: true }));
                        setInlineEdit({ path: fullPath, type: 'new-file', initialValue: '' });
                        setEditValue('');
                      } else {
                        window.dispatchEvent(new CustomEvent('open-file', { detail: { filepath: normalizedPath } }));
                      }
                    }},
                    ...(isDir ? [{ label: 'New Folder', action: () => {
                      setExpanded(prev => ({ ...prev, [fullPath]: true }));
                      setInlineEdit({ path: fullPath, type: 'new-dir', initialValue: '' });
                      setEditValue('');
                    }}] : []),
                    { label: 'Rename', action: () => {
                      setInlineEdit({ path: fullPath, type: 'rename', initialValue: node.name });
                      setEditValue(node.name);
                    }},
                    { divider: true },
                    { label: 'Delete', danger: true, action: () => {
                      if(confirm(`Are you sure you want to delete ${node.name}?`)) handleAction('delete', normalizedPath);
                    }},
                    { label: 'Copy Path', action: () => navigator.clipboard.writeText(normalizedPath) }
                  ]
                }
              }));
            }}
          >
            {isDir ? (
              <svg className={`w-3.5 h-3.5 text-text-muted transition-transform shrink-0 ${isExpanded ? 'rotate-90' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" /></svg>
            ) : (
              <span className="w-3.5 shrink-0" />
            )}
            
            {isDir ? (
              <svg className="w-4 h-4 text-accent-blue shrink-0" fill="currentColor" viewBox="0 0 20 20"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" /></svg>
            ) : (
              <svg className="w-4 h-4 text-text-muted shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>
            )}
            
            {inlineEdit?.path === fullPath && inlineEdit.type === 'rename' ? (
              <input 
                type="text" 
                autoFocus
                value={editValue}
                onChange={e => setEditValue(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') {
                    const newPath = normalizedPath.substring(0, normalizedPath.lastIndexOf(node.name)) + editValue;
                    handleAction('rename', normalizedPath, newPath);
                  }
                  if (e.key === 'Escape') setInlineEdit(null);
                }}
                onBlur={() => setInlineEdit(null)}
                className="bg-surface-base border border-accent-blue rounded px-1 text-text-primary text-xs w-full focus:outline-none"
              />
            ) : (
              <span className={`truncate ${isSelected ? 'font-semibold' : isDir ? 'font-semibold text-text-primary' : 'hover:text-text-primary'}`}>{node.name}</span>
            )}
            
            {!isDir && node.size !== undefined && inlineEdit?.path !== fullPath && (
              <span className="ml-auto text-[10px] text-text-muted shrink-0">{Math.round(node.size / 1024)}kb</span>
            )}
          </div>
          
          {isDir && isExpanded && (
            <div className="pl-4 border-l border-border-subtle ml-2 my-0.5">
              {(inlineEdit?.path === fullPath && inlineEdit.type !== 'rename') && (
                <div className="flex items-center gap-1.5 py-1 px-2">
                  <span className="w-3.5" />
                  {inlineEdit.type === 'new-dir' ? (
                    <svg className="w-4 h-4 text-accent-blue" fill="currentColor" viewBox="0 0 20 20"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" /></svg>
                  ) : (
                    <svg className="w-4 h-4 text-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>
                  )}
                  <input 
                    type="text" 
                    autoFocus
                    value={editValue}
                    onChange={e => setEditValue(e.target.value)}
                    onKeyDown={e => {
                      if (e.key === 'Enter') {
                        handleAction('create', normalizedPath ? `${normalizedPath}/${editValue}` : editValue, undefined, inlineEdit.type === 'new-dir');
                      }
                      if (e.key === 'Escape') setInlineEdit(null);
                    }}
                    onBlur={() => setInlineEdit(null)}
                    className="bg-surface-base border border-accent-blue rounded px-1 text-text-primary text-xs w-full focus:outline-none"
                  />
                </div>
              )}
              {node.children && renderTree(node.children, fullPath)}
            </div>
          )}
        </div>
      );
    });
  };

  return (
    <div className="w-full h-full bg-surface-base flex flex-col">
      <div className="p-2 border-b border-border-default bg-surface-raised shrink-0 flex items-center gap-2">
        <div className="relative flex-1">
          <svg className="w-4 h-4 text-text-muted absolute left-2.5 top-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
          <input 
            type="text" 
            placeholder="Search files..."
            className="w-full bg-surface-base border border-border-default rounded pl-8 pr-3 py-1.5 text-xs text-text-primary focus:outline-none focus:border-accent-blue"
          />
        </div>
        <button 
          className="p-1.5 text-text-muted hover:text-text-primary hover:bg-surface-overlay rounded transition-colors shrink-0"
          title="Explorer Settings"
          onClick={(e) => {
            e.stopPropagation();
            const rect = e.currentTarget.getBoundingClientRect();
            window.dispatchEvent(new CustomEvent('open-context-menu', {
              detail: {
                x: rect.right,
                y: rect.bottom + 4, // Dropdown effect
                items: [
                  { label: 'Collapse All', action: () => setExpanded({}) },
                  { label: 'Refresh', action: () => loadTree() },
                  { divider: true },
                  { label: 'Show Hidden Files', action: () => alert('Hidden files coming soon') }
                ]
              }
            }));
          }}
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
        </button>
      </div>
      <div 
        className="flex-1 overflow-y-auto p-2"
        onClick={() => {
          setSelectedFiles(new Set());
          setLastSelected('');
        }}
        onContextMenu={(e) => {
          e.preventDefault();
          window.dispatchEvent(new CustomEvent('open-context-menu', {
            detail: {
              x: e.clientX,
              y: e.clientY,
              items: [
                { label: 'New File', action: () => {
                  setInlineEdit({ path: '', type: 'new-file', initialValue: '' });
                  setEditValue('');
                }},
                { label: 'New Folder', action: () => {
                  setInlineEdit({ path: '', type: 'new-dir', initialValue: '' });
                  setEditValue('');
                }},
                { divider: true },
                { label: 'Refresh', action: () => loadTree() }
              ]
            }
          }));
        }}
      >
        {loading ? (
          <div className="flex justify-center items-center h-full text-text-muted">Loading...</div>
        ) : (
          <>
            {(inlineEdit?.path === '' && inlineEdit.type !== 'rename') && (
              <div className="flex items-center gap-1.5 py-1 px-2">
                {inlineEdit.type === 'new-dir' ? (
                  <svg className="w-4 h-4 text-accent-blue" fill="currentColor" viewBox="0 0 20 20"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" /></svg>
                ) : (
                  <svg className="w-4 h-4 text-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" /></svg>
                )}
                <input 
                  type="text" 
                  autoFocus
                  value={editValue}
                  onChange={e => setEditValue(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter') handleAction('create', editValue, undefined, inlineEdit.type === 'new-dir');
                    if (e.key === 'Escape') setInlineEdit(null);
                  }}
                  onBlur={() => setInlineEdit(null)}
                  className="bg-surface-base border border-accent-blue rounded px-1 text-text-primary text-xs w-full focus:outline-none"
                />
              </div>
            )}
            {renderTree(tree)}
          </>
        )}
      </div>
    </div>
  );
}
