import { useState, useEffect, useRef } from 'react';
import Editor from '@monaco-editor/react';
import { BACKEND_URL } from '../config';

const DEFAULT_CODE = `// AIOF Agent Configuration
export const orchestratorConfig = {
  version: "1.0.0",
  agents: [
    {
      id: "router",
      model: "gemini-3.1-pro-preview",
      capabilities: ["routing", "analysis"]
    },
    {
      id: "executor",
      model: "gemini-3.5-flash",
      capabilities: ["execution", "filesystem"]
    }
  ],
  telemetry: {
    enabled: true,
    level: "debug"
  }
};
`;

export default function CodeEditor({  }: { panelId: string }) {
  const [code, setCode] = useState(DEFAULT_CODE);
  const [filepath, setFilepath] = useState(() => localStorage.getItem('aiof-active-file') || 'config.ts');
  const [isRenaming, setIsRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  
  const editorRef = useRef<any>(null);

  useEffect(() => {
    const handleOpenFile = async (e: any) => {
      const path = e.detail.filepath;
      setFilepath(path);
      localStorage.setItem('aiof-active-file', path);
      try {
        const res = await fetch(`${BACKEND_URL}/api/read?filepath=${encodeURIComponent(path)}`);
        if (res.ok) {
          const text = await res.text();
          setCode(text);
        } else {
          console.error("Failed to read file");
        }
      } catch (err) {
        console.error(err);
      }
    };

    // Load initial file on mount if it's persisted
    const initialPath = localStorage.getItem('aiof-active-file');
    if (initialPath) {
      handleOpenFile({ detail: { filepath: initialPath } } as any);
    }

    window.addEventListener('open-file', handleOpenFile);
    return () => window.removeEventListener('open-file', handleOpenFile);
  }, []);

  const handleEditorDidMount = (editor: any) => {
    editorRef.current = editor;
    editor.onDidChangeCursorPosition((e: any) => {
      window.dispatchEvent(new CustomEvent('editor-cursor', {
        detail: { line: e.position.lineNumber, col: e.position.column }
      }));
    });
  };

  const handleSave = async () => {
    setSaving(true);
    setSaved(false);
    try {
      await fetch(`${BACKEND_URL}/api/save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ filepath, content: code })
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      console.error(e);
    } finally {
      setSaving(false);
    }
  };

  const handleRename = async () => {
    if (!renameValue || renameValue === filepath) {
      setIsRenaming(false);
      return;
    }
    try {
      const res = await fetch(`${BACKEND_URL}/api/rename`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ oldPath: filepath, newPath: renameValue })
      });
      if (res.ok) {
        setFilepath(renameValue);
        localStorage.setItem('aiof-active-file', renameValue);
        // Dispatch open-file to refresh tree selection and active file state
        window.dispatchEvent(new CustomEvent('open-file', { detail: { filepath: renameValue } }));
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsRenaming(false);
    }
  };

  return (
    <div className="w-full h-full flex flex-col bg-[#1e1e1e]">
      <div className="h-10 bg-surface-raised border-b border-border-default flex items-center px-4 justify-between shrink-0">
        <div 
          className="flex items-center gap-2 text-text-secondary text-sm font-mono cursor-pointer hover:text-text-primary transition-colors group"
          onClick={() => { setRenameValue(filepath); setIsRenaming(true); }}
          title="Click to rename"
        >
          <svg className="w-4 h-4 text-accent-amber" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
          {isRenaming ? (
            <input 
              autoFocus
              type="text" 
              value={renameValue}
              onChange={e => setRenameValue(e.target.value)}
              onBlur={handleRename}
              onKeyDown={e => {
                if (e.key === 'Enter') handleRename();
                if (e.key === 'Escape') setIsRenaming(false);
              }}
              className="bg-surface-overlay border border-accent-blue rounded px-1.5 py-0.5 text-text-primary text-xs focus:outline-none focus:ring-1 focus:ring-accent-blue"
            />
          ) : (
            <>
              {filepath}
              <svg className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
            </>
          )}
        </div>
        <button 
          onClick={handleSave}
          disabled={saving}
          className="px-3 py-1 bg-accent-blue/20 text-accent-blue hover:bg-accent-blue/30 rounded text-xs font-semibold transition-colors flex items-center gap-2 disabled:opacity-50"
        >
          {saving ? 'Saving...' : saved ? '✓ Saved' : 'Save'}
        </button>
      </div>
      
      <div className="flex-1 min-h-0">
        <Editor
          height="100%"
          defaultLanguage="typescript"
          theme="vs-dark"
          value={code}
          onChange={(val) => setCode(val || '')}
          onMount={handleEditorDidMount}
          options={{
            minimap: { enabled: true },
            fontSize: 13,
            fontFamily: '"JetBrains Mono", "Fira Code", monospace',
            wordWrap: 'on',
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            roundedSelection: false,
            padding: { top: 16 }
          }}
        />
      </div>
    </div>
  );
}
